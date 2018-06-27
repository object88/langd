package langd

import (
	"fmt"
	"go/ast"
	"go/build"
	"go/types"
	"sync"

	"github.com/object88/langd/collections"
	"github.com/pkg/errors"
)

// DistinctPackage contains the os/arch specific package AST
type DistinctPackage struct {
	Package   *Package
	hash      collections.Hash
	lc        LoaderContext
	loadState loadState

	m sync.Mutex
	c *sync.Cond

	buildPkg *build.Package
	checker  *types.Checker
	files    map[string]*File
	typesPkg *types.Package
}

// NewDistinctPackage returns a new instance of DistinctPackage
func NewDistinctPackage(lc LoaderContext, p *Package) *DistinctPackage {
	hash := lc.CalculateDistinctPackageHash(p.AbsPath)
	dp := &DistinctPackage{
		Package: p,
		files:   map[string]*File{},
		hash:    hash,
		lc:      lc,
	}
	dp.c = sync.NewCond(&dp.m)

	return dp
}

func (dp *DistinctPackage) check() error {
	if dp.checker == nil {
		info := &types.Info{
			Defs: map[*ast.Ident]types.Object{},
			// Implicits:  map[ast.Node]types.Object{},
			// Scopes:     map[ast.Node]*types.Scope{},
			Selections: map[*ast.SelectorExpr]*types.Selection{},
			// Types:      map[ast.Expr]types.TypeAndValue{},
			Uses: map[*ast.Ident]types.Object{},
		}

		dp.typesPkg = types.NewPackage(dp.Package.AbsPath, dp.buildPkg.Name)
		dp.checker = types.NewChecker(dp.lc.GetConfig(), dp.Package.Fset, dp.typesPkg, info)
	}

	// Loop over files and clear previous errors; all will be rechecked.
	astFiles := []*ast.File{}
	for _, v := range dp.files {
		f := v
		if !f.checked {
			f.errs = []FileError{}
			astFiles = append(astFiles, f.file)
		}
	}

	if len(astFiles) == 0 {
		return nil
	}

	dp.m.Lock()
	err := dp.checker.Files(astFiles)
	dp.m.Unlock()

	for _, v := range dp.files {
		v.checked = true
	}

	if err != nil {
		return errors.Wrapf(err, "DistinctPackage.check (%s): Checker failed", dp)
	}

	return nil
}

func (dp *DistinctPackage) GenerateBuildPackage(context *build.Context) error {
	buildPkg, err := context.Import(".", dp.Package.AbsPath, 0)
	if err != nil {
		if _, ok := err.(*build.NoGoError); ok {
			// There isn't any Go code here.
			return nil
		}
		return errors.Wrapf(err, "GenerateBuildPackage (%s): error while importing with build.Context", dp)
	}

	dp.buildPkg = buildPkg
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
