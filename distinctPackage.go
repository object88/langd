package langd

import (
	"fmt"
	"go/ast"
	"go/build"
	"go/types"
	"sync"

	"github.com/object88/langd/collections"
)

// DistinctPackage contains the os/arch specific package AST
type DistinctPackage struct {
	Package   *Package
	hash      collections.Hash
	lc        LoaderContext
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

// NewDistinctPackage returns a new instance of DistinctPackage
func NewDistinctPackage(lc LoaderContext, p *Package) *DistinctPackage {
	hash := lc.CalculateDistinctPackageHash(p.AbsPath)
	dp := &DistinctPackage{
		Package:         p,
		hash:            hash,
		lc:              lc,
		importPaths:     map[string]bool{},
		testImportPaths: map[string]bool{},
	}
	dp.c = sync.NewCond(&dp.m)

	return dp
}

func (dp *DistinctPackage) check() error {
	if dp.checker == nil {
		info := &types.Info{
			Defs:       map[*ast.Ident]types.Object{},
			Implicits:  map[ast.Node]types.Object{},
			Scopes:     map[ast.Node]*types.Scope{},
			Selections: map[*ast.SelectorExpr]*types.Selection{},
			Types:      map[ast.Expr]types.TypeAndValue{},
			Uses:       map[*ast.Ident]types.Object{},
		}

		dp.typesPkg = types.NewPackage(dp.Package.AbsPath, dp.buildPkg.Name)
		dp.checker = types.NewChecker(dp.lc.GetConfig(), dp.Package.Fset, dp.typesPkg, info)
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

	dp.m.Lock()
	err := dp.checker.Files(astFiles)
	dp.m.Unlock()
	return err
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

// Invalidate sets the checker to nil and the loadState to unloaded
func (dp *DistinctPackage) Invalidate() {
	dp.loadState = unloaded
	dp.checker = nil
	dp.typesPkg = nil
}

func (dp *DistinctPackage) String() string {
	return fmt.Sprintf("%s %s", dp.lc.GetTags(), dp.Package)
}

// WaitUntilReady blocks until this distinct package has loaded sufficiently
// for the requested load state.
func (dp *DistinctPackage) WaitUntilReady(loadState loadState) {
	check := func() bool {
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

	dp.m.Lock()
	for !check() {
		dp.c.Wait()
	}
	dp.m.Unlock()
}
