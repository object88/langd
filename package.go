package langd

import (
	"go/token"
	"sync"

	"github.com/OneOfOne/xxhash"
	"github.com/object88/langd/collections"
)

// BuildPackageHash returns the proper key for the provided path in the context of
// the LoaderContext's arch & os
func BuildPackageHash(absPath string) collections.Hash {
	h := xxhash.New64()
	h.WriteString(absPath)
	return collections.Hash(h.Sum64())
}

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
		hash:           BuildPackageHash(absPath),
		Fset:           token.NewFileSet(),
		fileHashes:     map[string]collections.Hash{},
		distincts:      map[collections.Hash]*DistinctPackage{},
		loaderContexts: map[LoaderContext]bool{},
	}

	return p
}

func (p *Package) EnsureDistinct(lc LoaderContext) (*DistinctPackage, bool) {
	created := false
	dhash := lc.GetDistinctHash()
	dp, ok := p.distincts[dhash]
	if !ok {
		dp = NewDistinctPackage(dhash, lc.GetContextArch(), lc.GetContextOS(), "")
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
	for _, dp := range p.distincts {
		dp.resetChecker()
	}
}

func (p *Package) String() string {
	return p.AbsPath
}
