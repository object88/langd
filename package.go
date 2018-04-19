package langd

import (
	"fmt"
	"go/build"
	"go/token"
	"go/types"
	"sync"

	"github.com/object88/langd/collections"
)

// Package is the contents of a package
type Package struct {
	key       collections.Key
	AbsPath   string
	GOARCH    string
	GOOS      string
	Tags      string
	shortPath string

	buildPkg *build.Package
	checker  *types.Checker
	Fset     *token.FileSet
	typesPkg *types.Package

	files           map[string]*File
	importPaths     map[string]bool
	testFiles       map[string]*File
	testImportPaths map[string]bool

	loadState loadState
	m         sync.Mutex
	c         *sync.Cond
}

// Key returns the collection key for the given Package
func (p *Package) Key() collections.Key {
	return p.key
}

// ResetChecker sets the checker to nil
func (p *Package) ResetChecker() {
	p.checker = nil
}

func (p *Package) String() string {
	return p.shortPath
}

func (p *Package) currentFiles() map[string]*File {
	loadState := p.loadState.get()
	switch loadState {
	case unloaded:
		if p.files == nil {
			p.files = map[string]*File{}
		}
		return p.files
	case loadedGo:
		if p.testFiles == nil {
			p.testFiles = map[string]*File{}
		}
		return p.testFiles
	}
	fmt.Printf("Package '%s' has loadState %d; no files.\n", p.AbsPath, loadState)
	return nil
}
