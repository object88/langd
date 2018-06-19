package langd

import (
	"fmt"
	"go/ast"
	"go/build"
	"go/types"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"unicode/utf8"

	"github.com/OneOfOne/xxhash"
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
	CheckPackage(p *Package) error
	EnsurePackage(absPath string) (*Package, *DistinctPackage, bool)
	FindImportPath(p *Package, importPath string) (string, error)
	GetContextArch() string
	GetContextOS() string
	GetDistinctHash() collections.Hash
	GetStartDir() string
	ImportBuildPackage(p *Package) *build.Package
	IsAllowed(absPath string) bool
	IsUnsafe(p *Package) bool

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

	packages map[collections.Hash]bool

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

	fmt.Printf("LC Start dir: %s\n", startDir)

	if strings.HasPrefix(startDir, "file://") {
		startDir = startDir[utf8.RuneCountInString("file://"):]
	}
	startDir, err := filepath.Abs(startDir)
	if err != nil {
		return nil
	}

	lc := &loaderContext{
		filteredPaths: globs,
		loader:        loader,
		packages:      map[collections.Hash]bool{},
		startDir:      startDir,
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

	lc.unsafePath = filepath.Join(lc.context.GOROOT, "src", "unsafe")

	c := &types.Config{
		Error:    lc.HandleTypeCheckerError,
		Importer: lc,
	}

	h := xxhash.New64()
	h.WriteString(goarch)
	h.WriteString(goos)
	h.WriteString(strings.Join(lc.Tags, ","))
	hash := collections.Hash(h.Sum64())

	lc.config = c
	lc.hash = hash

	return lc
}

func (lc *loaderContext) GetContextArch() string {
	return lc.context.GOARCH
}

func (lc *loaderContext) GetContextOS() string {
	return lc.context.GOOS
}

func (lc *loaderContext) GetDistinctHash() collections.Hash {
	return lc.hash
}

func (lc *loaderContext) AreAllPackagesComplete() bool {
	lc.m.Lock()

	if len(lc.packages) == 0 {
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
	dhash := lc.GetDistinctHash()
	for hash := range lc.packages {
		n, ok := caravan.Find(hash)
		if !ok {
			fmt.Printf("loaderContext.AreAllPackagesComplete (%s): package hash %x not found in caravan\n", lc, hash)
			complete = false
			break
		}
		p := n.Element.(*Package)
		dp, ok := p.distincts[dhash]
		if !ok {
			fmt.Printf("loaderContext.AreAllPackagesComplete (%s): distinct package for %s not found\n", lc, p)
			complete = false
			break
		}
		loadState := dp.loadState.get()
		if loadState != done {
			fmt.Printf("loaderContext.AreAllPackagesComplete (%s): distinct package for %s is not yet complete\n", lc, p)
			complete = false
			break
		}
	}

	fmt.Printf("loaderContext.AreAllPackagesComplete (%s): found to be complete? %t\n", lc, complete)
	lc.m.Unlock()
	return complete
}

func (lc *loaderContext) CheckPackage(p *Package) error {
	dp := p.distincts[lc.GetDistinctHash()]
	if dp.checker == nil {
		info := &types.Info{
			Defs:       map[*ast.Ident]types.Object{},
			Implicits:  map[ast.Node]types.Object{},
			Scopes:     map[ast.Node]*types.Scope{},
			Selections: map[*ast.SelectorExpr]*types.Selection{},
			Types:      map[ast.Expr]types.TypeAndValue{},
			Uses:       map[*ast.Ident]types.Object{},
		}

		dp.typesPkg = types.NewPackage(p.AbsPath, dp.buildPkg.Name)
		dp.checker = types.NewChecker(lc.config, p.Fset, dp.typesPkg, info)
	}

	// Loop over files and clear previous errors; all will be rechecked.
	files := dp.currentFiles()
	astFiles := make([]*ast.File, len(files))
	i := 0
	for _, v := range files {
		f := v
		f.errs = []FileError{}
		astFiles[i] = f.file
		i++
	}

	lc.checkerMu.Lock()
	err := dp.checker.Files(astFiles)
	lc.checkerMu.Unlock()
	return err
}

func (lc *loaderContext) EnsurePackage(absPath string) (*Package, *DistinctPackage, bool) {
	hash := BuildPackageHash(absPath)
	n, created := lc.loader.Caravan().Ensure(hash, func() collections.Hasher {
		lc.Log.Debugf("NewPackage: creating package for '%s'.\n", absPath)
		return NewPackage(absPath)
	})
	p := n.Element.(*Package)

	p.m.Lock()
	p.loaderContexts[lc] = true
	p.m.Unlock()

	lc.m.Lock()
	lc.packages[hash] = true
	lc.m.Unlock()

	dp, created := p.EnsureDistinct(lc)
	return p, dp, created
}

func (lc *loaderContext) FindImportPath(p *Package, importPath string) (string, error) {
	targetPath, err := lc.findImportPath(importPath, p.AbsPath)
	if err != nil {
		err := errors.Wrap(err, fmt.Sprintf("Failed to find import %s", importPath))
		return "", err
	}
	if targetPath == p.AbsPath {
		lc.Log.Debugf("Failed due to self-import\n")
		return "", err
	}

	return targetPath, nil
}

func (lc *loaderContext) GetStartDir() string {
	return lc.startDir
}

func (lc *loaderContext) ImportBuildPackage(p *Package) *build.Package {
	buildPkg, err := lc.context.Import(".", p.AbsPath, 0)
	if err != nil {
		if _, ok := err.(*build.NoGoError); ok {
			// There isn't any Go code here.
			return nil
		}
		lc.Log.Debugf("ImportBuildPackage: %s: proc error:\n\t%s\n", p, err.Error())
		return nil
	}

	return buildPkg
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
func (lc *loaderContext) IsUnsafe(p *Package) bool {
	return lc.unsafePath == p.AbsPath
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
	fmt.Printf("%s: Entering signal...\n", lc)
	lc.c.Broadcast()
	fmt.Printf("%s: Exiting signal...\n", lc)
}

func (lc *loaderContext) Wait() {
	if lc.AreAllPackagesComplete() {
		return
	}
	fmt.Printf("%s: Entering wait lock...\n", lc)
	lc.c.L.Lock()
	fmt.Printf("%s: Entering wait...\n", lc)
	lc.c.Wait()
	fmt.Printf("%s: Exiting wait...\n", lc)
	lc.c.L.Unlock()
	fmt.Printf("%s: Exiting wait lock...\n", lc)
}

// String is the implementation of fmt.Stringer
func (lc *loaderContext) String() string {
	x := make([]string, 2+len(lc.Tags))
	x[0] = lc.context.GOARCH
	x[1] = lc.context.GOOS
	for k, v := range lc.Tags {
		x[k+2] = v
	}
	return fmt.Sprintf("%s [%s]", lc.startDir, strings.Join(x, ", "))
}

// Import is the implementation of types.Importer
func (lc *loaderContext) Import(path string) (*types.Package, error) {
	p, err := lc.locatePackages(path)
	if err != nil {
		return nil, err
	}
	if p == nil {
		return nil, fmt.Errorf("Path parsed, but does not contain package %s", path)
	}

	dp := p.distincts[lc.GetDistinctHash()]

	return dp.typesPkg, nil
}

// ImportFrom is the implementation of types.ImporterFrom
func (lc *loaderContext) ImportFrom(path, srcDir string, mode types.ImportMode) (*types.Package, error) {
	absPath, err := lc.findImportPath(path, srcDir)
	if err != nil {
		fmt.Printf("Failed to locate import path for %s, %s:\n\t%s", path, srcDir, err.Error())
		return nil, fmt.Errorf("Failed to locate import path for %s, %s:\n\t%s", path, srcDir, err.Error())
	}

	p, err := lc.locatePackages(absPath)
	if err != nil {
		fmt.Printf("Failed to locate package %s\n\tfrom %s, %s:\n\t%s\n", absPath, path, srcDir, err.Error())
		return nil, err
	}

	dp := p.distincts[lc.hash]

	if dp.typesPkg == nil {
		fmt.Printf("\t%s (nil)\n", absPath)
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
		key := BuildPackageHash(absPath)
		node, ok := lc.loader.Caravan().Find(key)

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
		p := node.Element.(*Package)
		dp := p.distincts[lc.GetDistinctHash()]

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

func (lc *loaderContext) locatePackages(path string) (*Package, error) {
	n, ok := lc.loader.Caravan().Find(BuildPackageHash(path))
	if !ok {
		return nil, fmt.Errorf("Failed to import %s", path)
	}

	p := n.Element.(*Package)
	return p, nil
}
