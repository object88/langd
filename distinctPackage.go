package langd

import (
	"fmt"
	"go/build"
	"go/types"
	"sync"

	"github.com/object88/langd/collections"
)

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

	buildPkg *build.Package
	checker  *types.Checker
	typesPkg *types.Package
}

func NewDistinctPackage(hash collections.Hash, goarch, goos, tags string) *DistinctPackage {
	dp := &DistinctPackage{
		hash:            hash,
		GOARCH:          goarch,
		GOOS:            goos,
		Tags:            tags,
		importPaths:     map[string]bool{},
		testImportPaths: map[string]bool{},
	}
	dp.c = sync.NewCond(&dp.m)

	return dp
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
	// fmt.Printf("DistinctPackage has loadState %d; no files.\n", loadState)
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
