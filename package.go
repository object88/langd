package langd

import (
	"fmt"
	"go/build"
	"go/token"
	"go/types"
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
	hash      collections.Hash
	AbsPath   string
	shortPath string

	buildPkg *build.Package
	Fset     *token.FileSet

	// files           map[string]*File
	// importPaths     map[string]bool
	// testFiles       map[string]*File
	// testImportPaths map[string]bool

	// m sync.Mutex
	// c *sync.Cond

	distincts map[collections.Hash]*DistinctPackage

	loaderContexts map[LoaderContext]bool
}

func NewPackage(absPath, shortPath string, lc LoaderContext) *Package {
	p := &Package{
		AbsPath:   absPath,
		hash:      BuildPackageHash(absPath),
		shortPath: shortPath,
		Fset:      token.NewFileSet(),
		// importPaths:     map[string]bool{},
		// testImportPaths: map[string]bool{},
		distincts:      map[collections.Hash]*DistinctPackage{},
		loaderContexts: map[LoaderContext]bool{},
	}
	// p.c = sync.NewCond(&p.m)

	return p
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
	return p.shortPath
}

// DistinctPackage contains the os/arch specific package AST
type DistinctPackage struct {
	hash collections.Hash

	GOARCH string
	GOOS   string
	Tags   string

	loadState loadState

	files           map[string]*File
	importPaths     map[string]bool
	testFiles       map[string]*File
	testImportPaths map[string]bool

	m sync.Mutex
	c *sync.Cond

	checker  *types.Checker
	typesPkg *types.Package
}

func NewDistinctPackage(goarch, goos, tags string) {

}

func (dp *DistinctPackage) CheckReady(loadState loadState) bool {
	thisLoadState := dp.loadState.get()

	switch loadState {
	case queued:
		// Does not make sense that the source loadState would be here.
	case unloaded:
		return thisLoadState > unloaded
	case loadedGo:
		return thisLoadState > unloaded
	case loadedTest:
		// Should pass through here.
	default:
		// Should never get here.
	}

	return false
}

func (dp *DistinctPackage) currentFiles() map[string]*File {
	loadState := dp.loadState.get()
	switch loadState {
	case unloaded:
		if dp.files == nil {
			dp.files = map[string]*File{}
		}
		return dp.files
	case loadedGo:
		if dp.testFiles == nil {
			dp.testFiles = map[string]*File{}
		}
		return dp.testFiles
	}
	fmt.Printf("DistinctPackage has loadState %d; no files.\n", loadState)
	return nil
}

// Hash returns the hash for this distinct package
func (dp *DistinctPackage) Hash() collections.Hash {
	return dp.hash
}

// resetChecker sets the checker to nil and the loadState to unloaded
func (dp *DistinctPackage) resetChecker() {
	dp.loadState = unloaded
	dp.checker = nil
}

func (dp *DistinctPackage) String() string {
	return fmt.Sprintf("[%s, %s]", dp.GOARCH, dp.GOOS)
}
