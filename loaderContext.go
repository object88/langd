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

// LoaderContext is the workspace-specific configuration and context for
// building and type-checking
type LoaderContext interface {
	fmt.Stringer
	types.ImporterFrom

	AreAllPackagesComplete() bool
	CheckPackage(dp *DistinctPackage) error
	EnsureDistinctPackage(absPath string) (*DistinctPackage, bool)
	FindImportPath(dp *DistinctPackage, importPath string) (string, error)
	GetConfig() *types.Config
	GetDistinctHash() collections.Hash
	GetStartDir() string
	GetTags() string
	ImportBuildPackage(dp *DistinctPackage)
	IsAllowed(absPath string) bool
	IsUnsafe(dp *DistinctPackage) bool

	IsDir(absPath string) bool
	OpenFile(abdFilepath string) io.ReadCloser
	ReadDir(absPath string) ([]os.FileInfo, error)

	Signal()
	Wait()
}

type loaderContext struct {
	filteredPaths []glob.Glob
	hash          collections.Hash
	Tags          []string

	Log *log.Log

	loader Loader

	checkerMu  sync.Mutex
	config     *types.Config
	context    *build.Context
	startDir   string
	unsafePath string

	distinctPackageHashSet map[collections.Hash]bool

	m sync.Mutex
	c *sync.Cond
}

// LoaderContextOption provides a hook for NewLoaderContext to set or modify
// the new loader's build.Context
type LoaderContextOption func(lc LoaderContext)

// NewLoaderContext creates a new LoaderContext
func NewLoaderContext(loader Loader, startDir, goos, goarch, goroot string, options ...LoaderContextOption) LoaderContext {
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

	lc := &loaderContext{
		filteredPaths:          globs,
		loader:                 loader,
		distinctPackageHashSet: map[collections.Hash]bool{},
		startDir:               startDir,
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
		Importer: lc,
	}
	lc.hash = calculateHashFromStrings(append([]string{goarch, goos}, lc.Tags...)...)
	lc.unsafePath = filepath.Join(lc.context.GOROOT, "src", "unsafe")

	return lc
}

func (lc *loaderContext) GetConfig() *types.Config {
	return lc.config
}

func (lc *loaderContext) GetDistinctHash() collections.Hash {
	return lc.hash
}

func (lc *loaderContext) GetStartDir() string {
	return lc.startDir
}

func (lc *loaderContext) GetTags() string {
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

func (lc *loaderContext) AreAllPackagesComplete() bool {
	lc.m.Lock()

	if len(lc.distinctPackageHashSet) == 0 {
		// NOTE: this is a stopgap to address the problem where a loader context
		// will report that all packages are loaded before any of them have been
		// processed.  If we have a situation where a loader context is reading
		// a directory structure where there are legitimately no packages, this
		// will be a problem.
		fmt.Printf("loaderContext.AreAllPackagesComplete (%s): have zero packages\n", lc)
		lc.m.Unlock()
		return false
	}

	complete := true

	caravan := lc.loader.Caravan()
	for chash := range lc.distinctPackageHashSet {
		n, ok := caravan.Find(chash)
		if !ok {
			fmt.Printf("loaderContext.AreAllPackagesComplete (%s): package hash 0x%x not found in caravan\n", lc, chash)
			complete = false
			break
		}
		dp := n.Element.(*DistinctPackage)
		if !ok {
			fmt.Printf("loaderContext.AreAllPackagesComplete (%s): distinct package for %s not found\n", lc, dp)
			complete = false
			break
		}
		loadState := dp.loadState.get()
		if loadState != done {
			fmt.Printf("loaderContext.AreAllPackagesComplete (%s): distinct package for %s is not yet complete\n", lc, dp)
			complete = false
			break
		}
	}

	fmt.Printf("loaderContext.AreAllPackagesComplete (%s): found to be complete? %t\n", lc, complete)
	lc.m.Unlock()
	return complete
}

func (lc *loaderContext) CheckPackage(dp *DistinctPackage) error {
	lc.checkerMu.Lock()
	err := dp.check()
	lc.checkerMu.Unlock()
	return err
}

func (lc *loaderContext) EnsureDistinctPackage(absPath string) (*DistinctPackage, bool) {
	phash := calculateHashFromString(absPath)
	dhash := lc.GetDistinctHash()
	chash := combineHashes(phash, dhash)
	n, created := lc.loader.Caravan().Ensure(chash, func() collections.Hasher {
		lc.Log.Debugf("EnsureDistinctPackage: miss on hash 0x%x; creating package for '%s'.\n", chash, absPath)

		p, _ := lc.loader.EnsurePackage(absPath)
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

func (lc *loaderContext) FindImportPath(dp *DistinctPackage, importPath string) (string, error) {
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

func (lc *loaderContext) ImportBuildPackage(dp *DistinctPackage) {
	buildPkg, err := lc.context.Import(".", dp.Package.AbsPath, 0)
	if err != nil {
		if _, ok := err.(*build.NoGoError); ok {
			// There isn't any Go code here.
			return
		}
		lc.Log.Debugf("ImportBuildPackage: %s: proc error:\n\t%s\n", dp, err.Error())
		return
	}

	dp.buildPkg = buildPkg
}

func (lc *loaderContext) IsAllowed(absPath string) bool {
	for _, g := range lc.filteredPaths {
		if g.Match(absPath) {
			// We are looking at a filtered out path.
			return false
		}
	}

	return true
}

// IsUnsafe returns whether the provided package represents the `unsafe`
// package for the loader context
func (lc *loaderContext) IsUnsafe(dp *DistinctPackage) bool {
	return lc.unsafePath == dp.Package.AbsPath
}

func (lc *loaderContext) IsDir(absPath string) bool {
	return lc.context.IsDir(absPath)
}

func (lc *loaderContext) OpenFile(absFilepath string) io.ReadCloser {
	r, err := lc.context.OpenFile(absFilepath)
	if err != nil {
		lc.Log.Debugf("loaderContext.OpenFile: ERROR: Failed to open file %s:\n\t%s\n", absFilepath, err.Error())
		return nil
	}
	return r
}

func (lc *loaderContext) ReadDir(absPath string) ([]os.FileInfo, error) {
	return lc.context.ReadDir(absPath)
}

func (lc *loaderContext) Signal() {
	lc.c.Broadcast()
}

func (lc *loaderContext) Wait() {
	if lc.AreAllPackagesComplete() {
		return
	}
	lc.c.L.Lock()
	lc.c.Wait()
	lc.c.L.Unlock()
}

// String is the implementation of fmt.Stringer
func (lc *loaderContext) String() string {
	return fmt.Sprintf("%s %s", lc.startDir, lc.GetTags())
}

// Import is the implementation of types.Importer
func (lc *loaderContext) Import(path string) (*types.Package, error) {
	dp, err := lc.locatePackages(path)
	if err != nil {
		return nil, err
	}
	if dp == nil {
		return nil, fmt.Errorf("Path parsed, but does not contain package %s", path)
	}

	return dp.typesPkg, nil
}

// ImportFrom is the implementation of types.ImporterFrom
func (lc *loaderContext) ImportFrom(path, srcDir string, mode types.ImportMode) (*types.Package, error) {
	absPath, err := lc.findImportPath(path, srcDir)
	if err != nil {
		msg := fmt.Sprintf("Failed to locate import path for %s, %s", path, srcDir)
		return nil, errors.Wrap(err, msg)
	}

	dp, err := lc.locatePackages(absPath)
	if err != nil {
		msg := fmt.Sprintf("Failed to locate package %s\n\tfrom %s, %s", absPath, path, srcDir)
		return nil, errors.Wrap(err, msg)
	}

	if dp.typesPkg == nil {
		return nil, fmt.Errorf("Got nil in packages map")
	}

	return dp.typesPkg, nil
}

// HandleTypeCheckerError is invoked from the types.Checker when it encounters
// errors
func (lc *loaderContext) HandleTypeCheckerError(e error) {
	if terror, ok := e.(types.Error); ok {
		position := terror.Fset.Position(terror.Pos)
		absPath := filepath.Dir(position.Filename)
		phash := calculateHashFromString(absPath)
		dhash := lc.GetDistinctHash()
		chash := combineHashes(phash, dhash)
		node, ok := lc.loader.Caravan().Find(chash)

		if !ok {
			lc.Log.Debugf("ERROR: (missing) No package for %s\n", absPath)
			return
		}

		baseFilename := filepath.Base(position.Filename)
		ferr := FileError{
			Position: position,
			Message:  terror.Msg,
			Warning:  terror.Soft,
		}
		dp := node.Element.(*DistinctPackage)

		files := dp.currentFiles()
		f, ok := files[baseFilename]
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

func (lc *loaderContext) findImportPath(path, src string) (string, error) {
	buildPkg, err := lc.context.Import(path, src, build.FindOnly)
	if err != nil {
		msg := fmt.Sprintf("Failed to find import path:\n\tAttempted build.Import('%s', '%s', build.FindOnly)", path, src)
		return "", errors.Wrap(err, msg)
	}
	return buildPkg.Dir, nil
}

func (lc *loaderContext) locatePackages(path string) (*DistinctPackage, error) {
	dhash := lc.GetDistinctHash()
	phash := calculateHashFromString(path)
	chash := combineHashes(phash, dhash)
	n, ok := lc.loader.Caravan().Find(chash)
	if !ok {
		return nil, fmt.Errorf("Failed to import %s", path)
	}

	dp := n.Element.(*DistinctPackage)
	return dp, nil
}
