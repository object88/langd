package langd

import (
	"fmt"
	"go/ast"
	"go/build"
	"go/types"
	"sync"
)

// TODO:
// There is a problem with the current arrangement of Package and
// DistinctPackage, as it was assumed that Package would hold the references
// to DistinctPackages, and that the Caravan would hold references to the
// Package.  This will not work cleanly, however, because import may differ
// from the files in one DistinctPackage to another, causing a different
// DAG in the Caravan.
// This should be restructured so that the Caravan maps the relationship
// between DistinctPackages, and the DistinctPackage will embed a common
// Package to manage the shared resources.

// DistinctPackage contains the os/arch specific package AST
type DistinctPackage struct {
	lc LoaderContext
	p  *Package

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

func NewDistinctPackage(lc LoaderContext, p *Package) *DistinctPackage {
	dp := &DistinctPackage{
		lc:              lc,
		p:               p,
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

		dp.typesPkg = types.NewPackage(dp.p.AbsPath, dp.buildPkg.Name)
		dp.checker = types.NewChecker(dp.lc.GetConfig(), dp.p.Fset, dp.typesPkg, info)
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

// // Hash returns the hash for this distinct package
// func (dp *DistinctPackage) Hash() collections.Hash {
// 	return dp.lc.GetDistinctHash()
// }

// resetChecker sets the checker to nil and the loadState to unloaded
func (dp *DistinctPackage) resetChecker() {
	dp.loadState = unloaded
	dp.checker = nil
	dp.typesPkg = nil
}

func (dp *DistinctPackage) String() string {
	return fmt.Sprintf("[%s, %s]", dp.lc.GetContextArch(), dp.lc.GetContextOS())
}
