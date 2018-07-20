package langd

import (
	"go/token"
	"sync"

	"github.com/object88/langd/collections"
)

// Package is the contents of a package
type Package struct {
	AbsPath string

	fileHashes map[string]collections.Hash
	Fset       *token.FileSet

	m sync.Mutex

	loaders map[*Loader]bool
}

// NewPackage creates a new instance of a Package struct
func NewPackage(absPath string) *Package {
	p := &Package{
		AbsPath:    absPath,
		Fset:       token.NewFileSet(),
		fileHashes: map[string]collections.Hash{},
		loaders:    map[*Loader]bool{},
	}

	return p
}

// Invalidate resets the checker state for all distinct packages
func (p *Package) Invalidate() {
	p.Fset = token.NewFileSet()
}

func (p *Package) String() string {
	return p.AbsPath
}
