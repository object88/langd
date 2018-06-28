package langd

import (
	"go/token"
	"sync"

	"github.com/object88/langd/collections"
)

// Package is the contents of a package
type Package struct {
	hash    collections.Hash
	AbsPath string

	fileHashes map[string]collections.Hash
	Fset       *token.FileSet

	m sync.Mutex

	loaderContexts map[*LoaderContext]bool
}

// NewPackage creates a new instance of a Package struct
func NewPackage(absPath string) *Package {
	p := &Package{
		AbsPath:        absPath,
		hash:           calculateHashFromString(absPath),
		Fset:           token.NewFileSet(),
		fileHashes:     map[string]collections.Hash{},
		loaderContexts: map[*LoaderContext]bool{},
	}

	return p
}

// Hash returns the collection hash for the given Package
func (p *Package) Hash() collections.Hash {
	return p.hash
}

// Invalidate resets the checker state for all distinct packages
func (p *Package) Invalidate() {
	p.Fset = token.NewFileSet()
}

func (p *Package) String() string {
	return p.AbsPath
}
