package langd

import (
	"fmt"
	"go/build"
	"go/types"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"unicode/utf8"

	"github.com/gobwas/glob"
	"github.com/object88/langd/collections"
	"github.com/object88/langd/log"
	"github.com/pkg/errors"
)

// 	FindDistinctPackage(absPath string) (*DistinctPackage, error)
// 	GetTags() string

// 	IsDir(absPath string) bool
// 	OpenFile(abdFilepath string) io.ReadCloser
// 	ReadDir(absPath string) ([]os.FileInfo, error)

// LoaderContext is the workspace-specific configuration and context for
// building and type-checking
type LoaderContext struct {
	StartDir string

	filteredPaths []glob.Glob
	hash          collections.Hash
	Tags          []string

	Log *log.Log

	loader *Loader

	config     *types.Config
	context    *build.Context
	unsafePath string

	distinctPackageHashSet map[collections.Hash]bool

	m sync.Mutex
	c *sync.Cond
}

// LoaderContextOption provides a hook for NewLoaderContext to set or modify
// the new loader's build.Context
type LoaderContextOption func(lc *LoaderContext)

// NewLoaderContext creates a new LoaderContext
func NewLoaderContext(loader *Loader, startDir, goos, goarch, goroot string, options ...LoaderContextOption) *LoaderContext {
	globs := make([]glob.Glob, 2)
	globs[0] = glob.MustCompile(filepath.Join("**", ".*"))
	globs[1] = glob.MustCompile(filepath.Join("**", "testdata"))

	if strings.HasPrefix(startDir, "file://") {
		startDir = startDir[utf8.RuneCountInString("file://"):]
	}
	startDir, err := filepath.Abs(startDir)
	if err != nil {
		return nil
	}

	lc := &LoaderContext{
		StartDir:               startDir,
		filteredPaths:          globs,
		loader:                 loader,
		distinctPackageHashSet: map[collections.Hash]bool{},
	}

	lc.c = sync.NewCond(&lc.m)

	for _, opt := range options {
		opt(lc)
	}

	if lc.context == nil {
		lc.context = &build.Default
	}

	lc.context.GOARCH = goarch
	lc.context.GOOS = goos
	lc.context.GOROOT = goroot

	if lc.context.IsDir == nil {
		lc.context.IsDir = func(path string) bool {
			fi, err := os.Stat(path)
			return err == nil && fi.IsDir()
		}
	}
	if lc.context.OpenFile == nil {
		lc.context.OpenFile = func(path string) (io.ReadCloser, error) {
			f, err := os.Open(path)
			if err != nil {
				return nil, err
			}
			return f, nil
		}
	}
	if lc.context.ReadDir == nil {
		lc.context.ReadDir = func(dir string) ([]os.FileInfo, error) {
			return ioutil.ReadDir(dir)
		}
	}

	lc.config = &types.Config{
		Error:    lc.HandleTypeCheckerError,
		Importer: &loaderContextImporter{lc: lc},
	}
	lc.hash = calculateHashFromStrings(append([]string{goarch, goos}, lc.Tags...)...)
	lc.unsafePath = filepath.Join(lc.context.GOROOT, "src", "unsafe")

	return lc
}

func (lc *LoaderContext) calculateDistinctPackageHash(absPath string) collections.Hash {
	phash := calculateHashFromString(absPath)
	chash := combineHashes(phash, lc.hash)
	return chash
}

func (lc *LoaderContext) GetTags() string {
	var sb strings.Builder
	sb.WriteRune('[')
	sb.WriteString(lc.context.GOARCH)
	sb.WriteRune(',')
	sb.WriteString(lc.context.GOOS)
	for _, v := range lc.Tags {
		sb.WriteRune(',')
		sb.WriteString(v)
	}
	sb.WriteRune(']')
	return sb.String()
}

func (lc *LoaderContext) areAllPackagesComplete() bool {
	lc.m.Lock()

	if len(lc.distinctPackageHashSet) == 0 {
		// NOTE: this is a stopgap to address the problem where a loader context
		// will report that all packages are loaded before any of them have been
		// processed.  If we have a situation where a loader context is reading
		// a directory structure where there are legitimately no packages, this
		// will be a problem.
		fmt.Printf("loaderContext.areAllPackagesComplete (%s): have zero packages\n", lc)
		lc.m.Unlock()
		return false
	}

	complete := true

	caravan := lc.loader.caravan
	for chash := range lc.distinctPackageHashSet {
		n, ok := caravan.Find(chash)
		if !ok {
			fmt.Printf("loaderContext.areAllPackagesComplete (%s): package hash 0x%x not found in caravan\n", lc, chash)
			complete = false
			break
		}
		dp := n.Element.(*DistinctPackage)
		if !ok {
			fmt.Printf("loaderContext.areAllPackagesComplete (%s): distinct package for %s not found\n", lc, dp)
			complete = false
			break
		}
		loadState := dp.loadState.get()
		if loadState != done {
			complete = false
			break
		}
	}

	lc.m.Unlock()
	return complete
}

func (lc *LoaderContext) checkPackage(dp *DistinctPackage) error {
	lc.m.Lock()
	err := dp.check()
	lc.m.Unlock()
	return err
}

func (lc *LoaderContext) ensureDistinctPackage(absPath string) (*DistinctPackage, bool) {
	chash := lc.calculateDistinctPackageHash(absPath)
	n, created := lc.loader.caravan.Ensure(chash, func() collections.Hasher {
		lc.Log.Debugf("ensureDistinctPackage: miss on hash 0x%x; creating package for '%s'.\n", chash, absPath)

		p, _ := lc.loader.ensurePackage(absPath)
		return NewDistinctPackage(lc, p)
	})
	dp := n.Element.(*DistinctPackage)

	dp.m.Lock()
	dp.Package.loaderContexts[lc] = true
	dp.m.Unlock()

	lc.m.Lock()
	lc.distinctPackageHashSet[chash] = true
	lc.m.Unlock()

	return dp, created
}

// FindDistinctPackage will locate the distinct package at the provided path
func (lc *LoaderContext) FindDistinctPackage(absPath string) (*DistinctPackage, error) {
	chash := lc.calculateDistinctPackageHash(absPath)
	n, ok := lc.loader.caravan.Find(chash)
	if !ok {
		return nil, errors.Errorf("Loader does not have an entry for %s with tags %s", absPath, lc.GetTags())
	}
	dp := n.Element.(*DistinctPackage)
	return dp, nil
}

func (lc *LoaderContext) FindImportPath(dp *DistinctPackage, importPath string) (string, error) {
	targetPath, err := lc.findImportPath(importPath, dp.Package.AbsPath)
	if err != nil {
		err := errors.Wrap(err, fmt.Sprintf("Failed to find import %s", importPath))
		return "", err
	}
	if targetPath == dp.Package.AbsPath {
		lc.Log.Debugf("Failed due to self-import\n")
		return "", err
	}

	return targetPath, nil
}

func (lc *LoaderContext) isAllowed(absPath string) bool {
	for _, g := range lc.filteredPaths {
		if g.Match(absPath) {
			// We are looking at a filtered out path.
			return false
		}
	}

	return true
}

// isUnsafe returns whether the provided package represents the `unsafe`
// package for the loader context
func (lc *LoaderContext) isUnsafe(dp *DistinctPackage) bool {
	return lc.unsafePath == dp.Package.AbsPath
}

func (lc *LoaderContext) IsDir(absPath string) bool {
	return lc.context.IsDir(absPath)
}

func (lc *LoaderContext) OpenFile(absFilepath string) io.ReadCloser {
	r, err := lc.context.OpenFile(absFilepath)
	if err != nil {
		lc.Log.Debugf("loaderContext.OpenFile: ERROR: Failed to open file %s:\n\t%s\n", absFilepath, err.Error())
		return nil
	}
	return r
}

func (lc *LoaderContext) ReadDir(absPath string) ([]os.FileInfo, error) {
	return lc.context.ReadDir(absPath)
}

func (lc *LoaderContext) Signal() {
	lc.c.Broadcast()
}

// Wait blocks until all packages have been loaded
func (lc *LoaderContext) Wait() {
	if lc.areAllPackagesComplete() {
		return
	}
	lc.c.L.Lock()
	lc.c.Wait()
	lc.c.L.Unlock()
}

// String is the implementation of fmt.Stringer
func (lc *LoaderContext) String() string {
	return fmt.Sprintf("%s %s", lc.StartDir, lc.GetTags())
}

// HandleTypeCheckerError is invoked from the types.Checker when it encounters
// errors
func (lc *LoaderContext) HandleTypeCheckerError(e error) {
	if terror, ok := e.(types.Error); ok {
		position := terror.Fset.Position(terror.Pos)
		absPath := filepath.Dir(position.Filename)
		dp, err := lc.FindDistinctPackage(absPath)
		if err != nil {
			lc.Log.Debugf("ERROR: (missing) No package for %s\n\t%s\n", absPath, err.Error())
			return
		}

		baseFilename := filepath.Base(position.Filename)
		ferr := FileError{
			Position: position,
			Message:  terror.Msg,
			Warning:  terror.Soft,
		}

		f, ok := dp.files[baseFilename]
		if !ok {
			lc.Log.Debugf("ERROR: (missing file) %s\n", position.Filename)
		} else {
			f.errs = append(f.errs, ferr)
			lc.Log.Debugf("ERROR: (types error) %s\n", terror.Error())
		}
	} else {
		lc.Log.Debugf("ERROR: (unknown) %#v\n", e)
	}
}

func (lc *LoaderContext) findImportPath(path, src string) (string, error) {
	buildPkg, err := lc.context.Import(path, src, build.FindOnly)
	if err != nil {
		msg := fmt.Sprintf("Failed to find import path:\n\tAttempted build.Import('%s', '%s', build.FindOnly)", path, src)
		return "", errors.Wrap(err, msg)
	}
	return buildPkg.Dir, nil
}
