package langd

import (
	"fmt"
	"go/ast"
	"go/build"
	"go/token"
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
	types.ImporterFrom

	BuildKey(absPath string) collections.Key
	CheckPackage(p *Package) error
	FindImportPath(p *Package, importPath string) (string, error)
	GetStartDir() string
	ImportBuildPackage(p *Package) *build.Package
	IsAllowed(absPath string) bool
	IsDir(absPath string) bool
	IsUnsafe(p *Package) bool
	NewPackage(key collections.Key, absPath string) *Package
	OpenFile(abdFilepath string) io.ReadCloser
	ReadDir(absPath string) ([]os.FileInfo, error)

	Signal()
	Wait()
}

type loaderContext struct {
	filteredPaths []glob.Glob
	Tags          []string

	Log *log.Log

	loader Loader

	checkerMu  sync.Mutex
	config     *types.Config
	context    *build.Context
	startDir   string
	unsafePath string

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

	lc := &loaderContext{
		filteredPaths: globs,
		loader:        loader,
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

	lc.config = c

	return lc
}

// BuildKey returns the proper key for the provided path in the context of
// the LoaderContext's arch & os
func (lc *loaderContext) BuildKey(absPath string) collections.Key {
	h := xxhash.New64()
	h.WriteString(lc.context.GOARCH)
	h.WriteString(lc.context.GOOS)
	h.WriteString(absPath)
	return collections.Key(h.Sum64())
}

func (lc *loaderContext) CheckPackage(p *Package) error {
	if p.checker == nil {
		info := &types.Info{
			Defs:       map[*ast.Ident]types.Object{},
			Implicits:  map[ast.Node]types.Object{},
			Scopes:     map[ast.Node]*types.Scope{},
			Selections: map[*ast.SelectorExpr]*types.Selection{},
			Types:      map[ast.Expr]types.TypeAndValue{},
			Uses:       map[*ast.Ident]types.Object{},
		}

		p.typesPkg = types.NewPackage(p.AbsPath, p.buildPkg.Name)
		p.checker = types.NewChecker(lc.config, p.Fset, p.typesPkg, info)
	}

	// Clear previous errors; all will be rechecked.
	files := p.currentFiles()

	// Loop over packages
	astFiles := make([]*ast.File, len(files))
	i := 0
	for _, v := range files {
		f := v
		f.errs = []FileError{}
		astFiles[i] = f.file
		i++
	}

	lc.checkerMu.Lock()
	err := p.checker.Files(astFiles)
	lc.checkerMu.Unlock()
	return err
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

func (lc *loaderContext) IsDir(absPath string) bool {
	return lc.context.IsDir(absPath)
}

// IsUnsafe returns whether the provided package represents the `unsafe`
// package for the loader context
func (lc *loaderContext) IsUnsafe(p *Package) bool {
	return lc.unsafePath == p.AbsPath
}

func (lc *loaderContext) NewPackage(key collections.Key, absPath string) *Package {
	shortPath := absPath
	if strings.HasPrefix(absPath, lc.context.GOROOT) {
		shortPath = fmt.Sprintf("(%s, %s, stdlib) %s", lc.context.GOARCH, lc.context.GOOS, absPath[utf8.RuneCountInString(lc.context.GOROOT)+5:])
	} else {
		// Shorten the canonical name for logging purposes.
		n := utf8.RuneCountInString(lc.startDir)
		if len(absPath) >= n {
			shortPath = absPath[n:]
		}
		shortPath = fmt.Sprintf("(%s, %s) %s", lc.context.GOARCH, lc.context.GOOS, shortPath)
	}
	p := &Package{
		AbsPath:         absPath,
		GOARCH:          lc.context.GOARCH,
		GOOS:            lc.context.GOOS,
		key:             key,
		shortPath:       shortPath,
		Fset:            token.NewFileSet(),
		importPaths:     map[string]bool{},
		testImportPaths: map[string]bool{},
	}
	p.c = sync.NewCond(&p.m)

	lc.Log.Debugf("ensurePackage: creating package for '%s' at 0x%x.\n", p.String(), p.Key())
	return p
}

func (lc *loaderContext) OpenFile(absFilepath string) io.ReadCloser {
	r, err := lc.context.OpenFile(absFilepath)
	if err != nil {
		lc.Log.Debugf(" GF: ERROR: Failed to read file %s:\n\t%s\n", absFilepath, err.Error())
		return nil
	}
	return r
}

func (lc *loaderContext) ReadDir(absPath string) ([]os.FileInfo, error) {
	return lc.context.ReadDir(absPath)
}

func (lc *loaderContext) Signal() {
	fmt.Printf("Entering signal...\n")
	lc.c.Broadcast()
	fmt.Printf("Exiting signal...\n")
}

func (lc *loaderContext) Wait() {
	fmt.Printf("Entering wait lock...\n")
	lc.c.L.Lock()
	fmt.Printf("Entering wait...\n")
	lc.c.Wait()
	fmt.Printf("Exiting wait...\n")
	lc.c.L.Unlock()
	fmt.Printf("Exiting wait lock...\n")
}

// Import is the implementation of types.Importer
func (lc *loaderContext) Import(path string) (*types.Package, error) {
	// fmt.Printf("Importer looking for '%s'\n", path)
	p, err := lc.locatePackages(path)
	if err != nil {
		return nil, err
	}
	if p == nil {
		return nil, fmt.Errorf("Path parsed, but does not contain package %s", path)
	}

	return p.typesPkg, nil
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

	if p.typesPkg == nil {
		fmt.Printf("\t%s (nil)\n", absPath)
		return nil, fmt.Errorf("Got nil in packages map")
	}

	return p.typesPkg, nil
}

// HandleTypeCheckerError is invoked from the types.Checker when it encounters
// errors
func (lc *loaderContext) HandleTypeCheckerError(e error) {
	if terror, ok := e.(types.Error); ok {
		position := terror.Fset.Position(terror.Pos)
		absPath := filepath.Dir(position.Filename)
		key := lc.BuildKey(absPath)
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

		files := p.currentFiles()
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
		msg := fmt.Sprintf("Failed to find import path:\n\tAttempted build.Import('%s', '%s', build.FindOnly)\n\t%s\n", path, src, err.Error())
		return "", errors.New(msg)
	}
	return buildPkg.Dir, nil
}

func (lc *loaderContext) locatePackages(path string) (*Package, error) {
	n, ok := lc.loader.Caravan().Find(lc.BuildKey(path))
	if !ok {
		fmt.Printf("**** Not found! *****\n")
		return nil, fmt.Errorf("Failed to import %s", path)
	}

	p := n.Element.(*Package)

	return p, nil
}
