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

// Loader is the workspace-specific configuration and context for
// building and type-checking
type Loader struct {
	StartDir string

	filteredPaths []glob.Glob
	hash          collections.Hash
	Tags          []string

	Log *log.Log

	le *LoaderEngine

	config     *types.Config
	context    *build.Context
	unsafePath string

	distinctPackageHashSet map[collections.Hash]bool

	m sync.Mutex
	c *sync.Cond
}

// LoaderOption provides a hook for NewLoader to set or modify
// the new loader's build.Context
type LoaderOption func(l *Loader)

// NewLoader creates a new Loader
func NewLoader(le *LoaderEngine, startDir, goos, goarch, goroot string, options ...LoaderOption) *Loader {
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

	l := &Loader{
		StartDir:      startDir,
		filteredPaths: globs,
		le:            le,
		distinctPackageHashSet: map[collections.Hash]bool{},
	}

	l.c = sync.NewCond(&l.m)

	for _, opt := range options {
		opt(l)
	}

	if l.context == nil {
		l.context = &build.Default
	}

	l.context.GOARCH = goarch
	l.context.GOOS = goos
	l.context.GOROOT = goroot

	if l.context.IsDir == nil {
		l.context.IsDir = func(path string) bool {
			fi, err := os.Stat(path)
			return err == nil && fi.IsDir()
		}
	}
	if l.context.OpenFile == nil {
		l.context.OpenFile = func(path string) (io.ReadCloser, error) {
			f, err := os.Open(path)
			if err != nil {
				return nil, err
			}
			return f, nil
		}
	}
	if l.context.ReadDir == nil {
		l.context.ReadDir = func(dir string) ([]os.FileInfo, error) {
			return ioutil.ReadDir(dir)
		}
	}

	l.config = &types.Config{
		Error:    l.HandleTypeCheckerError,
		Importer: &loaderImporter{l: l},
	}
	l.hash = calculateHashFromStrings(append([]string{goarch, goos}, l.Tags...)...)
	l.unsafePath = filepath.Join(l.context.GOROOT, "src", "unsafe")

	return l
}

// Errors exposes problems with code found during compilation on a file-by-file
// basis.
func (l *Loader) Errors(handleErrs func(file string, errs []FileError)) {
	for hash := range l.distinctPackageHashSet {
		n, ok := l.le.caravan.Find(hash)
		if !ok {
			// TODO: This is probably a poor way of handling this problem.  The error
			// will bubble up to the user, who will have no idea what the hash means.
			errs := []FileError{
				FileError{
					Message: fmt.Sprintf("Failed to find node in caravan with hash 0x%x", hash),
					Warning: false,
				},
			}
			handleErrs("", errs)
			continue
		}

		dp := n.Element.(*DistinctPackage)
		for fname, f := range dp.files {
			if len(f.errs) != 0 {
				handleErrs(filepath.Join(dp.Package.AbsPath, fname), f.errs)
			}
		}
	}
}

func (l *Loader) calculateDistinctPackageHash(absPath string) collections.Hash {
	phash := calculateHashFromString(absPath)
	chash := combineHashes(phash, l.hash)
	return chash
}

func (l *Loader) GetTags() string {
	var sb strings.Builder
	sb.WriteRune('[')
	sb.WriteString(l.context.GOARCH)
	sb.WriteRune(',')
	sb.WriteString(l.context.GOOS)
	for _, v := range l.Tags {
		sb.WriteRune(',')
		sb.WriteString(v)
	}
	sb.WriteRune(']')
	return sb.String()
}

func (l *Loader) areAllPackagesComplete() bool {
	l.m.Lock()

	if len(l.distinctPackageHashSet) == 0 {
		// NOTE: this is a stopgap to address the problem where a loader context
		// will report that all packages are loaded before any of them have been
		// processed.  If we have a situation where a loader context is reading
		// a directory structure where there are legitimately no packages, this
		// will be a problem.
		fmt.Printf("loader.areAllPackagesComplete (%s): have zero packages\n", l)
		l.m.Unlock()
		return false
	}

	complete := true

	caravan := l.le.caravan
	for chash := range l.distinctPackageHashSet {
		n, ok := caravan.Find(chash)
		if !ok {
			fmt.Printf("loader.areAllPackagesComplete (%s): package hash 0x%x not found in caravan\n", l, chash)
			complete = false
			break
		}
		dp := n.Element.(*DistinctPackage)
		if !ok {
			fmt.Printf("loader.areAllPackagesComplete (%s): distinct package for %s not found\n", l, dp)
			complete = false
			break
		}
		loadState := dp.loadState.get()
		if loadState != done {
			complete = false
			break
		}
	}

	l.m.Unlock()
	return complete
}

func (l *Loader) checkPackage(dp *DistinctPackage) error {
	l.m.Lock()
	err := dp.check()
	l.m.Unlock()
	return err
}

func (l *Loader) ensureDistinctPackage(absPath string) (*DistinctPackage, bool) {
	chash := l.calculateDistinctPackageHash(absPath)
	n, created := l.le.caravan.Ensure(chash, func() collections.Hasher {
		l.Log.Debugf("ensureDistinctPackage: miss on hash 0x%x; creating package for '%s'.\n", chash, absPath)

		p, _ := l.le.ensurePackage(absPath)
		return NewDistinctPackage(l, p)
	})
	dp := n.Element.(*DistinctPackage)

	dp.m.Lock()
	dp.Package.loaders[l] = true
	dp.m.Unlock()

	l.m.Lock()
	l.distinctPackageHashSet[chash] = true
	l.m.Unlock()

	return dp, created
}

// FindDistinctPackage will locate the distinct package at the provided path
func (l *Loader) FindDistinctPackage(absPath string) (*DistinctPackage, error) {
	chash := l.calculateDistinctPackageHash(absPath)
	n, ok := l.le.caravan.Find(chash)
	if !ok {
		return nil, errors.Errorf("Loader does not have an entry for %s with tags %s", absPath, l.GetTags())
	}
	dp := n.Element.(*DistinctPackage)
	return dp, nil
}

func (l *Loader) FindImportPath(dp *DistinctPackage, importPath string) (string, error) {
	targetPath, err := l.findImportPath(importPath, dp.Package.AbsPath)
	if err != nil {
		err := errors.Wrap(err, fmt.Sprintf("Failed to find import %s", importPath))
		return "", err
	}
	if targetPath == dp.Package.AbsPath {
		l.Log.Debugf("Failed due to self-import\n")
		return "", err
	}

	return targetPath, nil
}

// LoadDirectory adds the contents of a directory to the Loader
func (l *Loader) LoadDirectory(path string) error {
	if !l.context.IsDir(path) {
		return fmt.Errorf("Argument '%s' is not a directory", path)
	}

	absPath, err := filepath.Abs(path)
	if err != nil {
		return fmt.Errorf("Could not get absolute path for '%s'", absPath)
	}

	l.Log.Verbosef("Loader.LoadDirectory: reading dir '%s'\n", absPath)
	l.le.readDir(l, absPath)

	return nil
}

func (l *Loader) isAllowed(absPath string) bool {
	for _, g := range l.filteredPaths {
		if g.Match(absPath) {
			// We are looking at a filtered out path.
			return false
		}
	}

	return true
}

// isUnsafe returns whether the provided package represents the `unsafe`
// package for the loader context
func (l *Loader) isUnsafe(dp *DistinctPackage) bool {
	return l.unsafePath == dp.Package.AbsPath
}

func (l *Loader) Signal() {
	l.c.Broadcast()
}

// Wait blocks until all packages have been loaded
func (l *Loader) Wait() {
	if l.areAllPackagesComplete() {
		return
	}
	l.c.L.Lock()
	l.c.Wait()
	l.c.L.Unlock()
}

// String is the implementation of fmt.Stringer
func (l *Loader) String() string {
	return fmt.Sprintf("%s %s", l.StartDir, l.GetTags())
}

// HandleTypeCheckerError is invoked from the types.Checker when it encounters
// errors
func (l *Loader) HandleTypeCheckerError(e error) {
	if terror, ok := e.(types.Error); ok {
		position := terror.Fset.Position(terror.Pos)
		absPath := filepath.Dir(position.Filename)
		dp, err := l.FindDistinctPackage(absPath)
		if err != nil {
			l.Log.Debugf("ERROR: (missing) No package for %s\n\t%s\n", absPath, err.Error())
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
			l.Log.Debugf("ERROR: (missing file) %s\n", position.Filename)
		} else {
			f.errs = append(f.errs, ferr)
			l.Log.Debugf("ERROR: (types error) %s\n", terror.Error())
		}
	} else {
		l.Log.Debugf("ERROR: (unknown) %#v\n", e)
	}
}

func (l *Loader) findImportPath(path, src string) (string, error) {
	buildPkg, err := l.context.Import(path, src, build.FindOnly)
	if err != nil {
		msg := fmt.Sprintf("Failed to find import path:\n\tAttempted build.Import('%s', '%s', build.FindOnly)", path, src)
		return "", errors.Wrap(err, msg)
	}
	return buildPkg.Dir, nil
}
