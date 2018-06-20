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

	distincts map[collections.Hash]*DistinctPackage

	loaderContexts map[LoaderContext]bool
}

// NewPackage creates a new instance of a Package struct
func NewPackage(absPath string) *Package {
	p := &Package{
		AbsPath:        absPath,
		hash:           calculateHashFromString(absPath),
		Fset:           token.NewFileSet(),
		fileHashes:     map[string]collections.Hash{},
		distincts:      map[collections.Hash]*DistinctPackage{},
		loaderContexts: map[LoaderContext]bool{},
	}

	return p
}

// EnsureDistinct checks for a distinct package on this package that matches
// the provided LoaderContext, and if missing, will create it
func (p *Package) EnsureDistinct(lc LoaderContext) (*DistinctPackage, bool) {
	created := false
	dhash := lc.GetDistinctHash()
	dp, ok := p.distincts[dhash]
	if !ok {
		dp = NewDistinctPackage(lc, p)
		p.distincts[dhash] = dp
		created = true
	}

	return dp, created
}

// Hash returns the collection hash for the given Package
func (p *Package) Hash() collections.Hash {
	return p.hash
}

// Invalidate resets the checker state for all distinct packages
func (p *Package) Invalidate() {
	p.Fset = token.NewFileSet()
	for _, dp := range p.distincts {
		dp.resetChecker()
	}
}

func (p *Package) String() string {
	return p.AbsPath
}
