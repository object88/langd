package langd

import (
	"errors"
	"fmt"
	"go/build"
	"go/types"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"sync"

	"github.com/OneOfOne/xxhash"
	"github.com/gobwas/glob"
	"github.com/object88/langd/collections"
	"github.com/object88/langd/log"
)

// LoaderContext is the workspace-specific configuration and context for
// building and type-checking
type LoaderContext struct {
	filteredPaths []glob.Glob
	Tags          []string

	Log *log.Log

	loader *Loader

	checkerMu  sync.Mutex
	config     *types.Config
	context    *build.Context
	startDir   string
	unsafePath string
}

// LoaderContextOption provides a hook for NewLoaderContext to set or modify
// the new loader's build.Context
type LoaderContextOption func(lc *LoaderContext)

// NewLoaderContext creates a new LoaderContext
func NewLoaderContext(loader *Loader, goos, goarch, goroot string, options ...LoaderContextOption) *LoaderContext {
	globs := make([]glob.Glob, 2)
	globs[0] = glob.MustCompile(filepath.Join("**", ".*"))
	globs[1] = glob.MustCompile(filepath.Join("**", "testdata"))

	lc := &LoaderContext{
		filteredPaths: globs,
		loader:        loader,
	}

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
func (lc *LoaderContext) BuildKey(absPath string) collections.Key {
	h := xxhash.New64()
	h.WriteString(lc.context.GOARCH)
	h.WriteString(lc.context.GOOS)
	h.WriteString(absPath)
	return collections.Key(h.Sum64())
}

// IsUnsafe returns whether the provided package represents the `unsafe`
// package for the loader context
func (lc *LoaderContext) IsUnsafe(p *Package) bool {
	return lc.unsafePath == p.AbsPath
}

// Import is the implementation of types.Importer
func (lc *LoaderContext) Import(path string) (*types.Package, error) {
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
func (lc *LoaderContext) ImportFrom(path, srcDir string, mode types.ImportMode) (*types.Package, error) {
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
func (lc *LoaderContext) HandleTypeCheckerError(e error) {
	if terror, ok := e.(types.Error); ok {
		position := terror.Fset.Position(terror.Pos)
		absPath := filepath.Dir(position.Filename)
		key := lc.BuildKey(absPath)
		node, ok := lc.loader.caravan.Find(key)

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

func (lc *LoaderContext) findImportPath(path, src string) (string, error) {
	buildPkg, err := lc.context.Import(path, src, build.FindOnly)
	if err != nil {
		msg := fmt.Sprintf("Failed to find import path:\n\tAttempted build.Import('%s', '%s', build.FindOnly)\n\t%s\n", path, src, err.Error())
		return "", errors.New(msg)
	}
	return buildPkg.Dir, nil
}

func (lc *LoaderContext) locatePackages(path string) (*Package, error) {
	n, ok := lc.loader.caravan.Find(lc.BuildKey(path))
	if !ok {
		fmt.Printf("**** Not found! *****\n")
		return nil, fmt.Errorf("Failed to import %s", path)
	}

	p := n.Element.(*Package)

	return p, nil
}
