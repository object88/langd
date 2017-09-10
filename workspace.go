package langd

import (
	"go/ast"
	"go/token"
	"go/types"
)

// Workspace is a mass of code
type Workspace struct {
	Fset     *token.FileSet
	Info     *types.Info
	PkgNames map[string]bool
	Files    map[string]*ast.File
}

func newWorkspace(fset *token.FileSet, info *types.Info, pkgNames map[string]bool, files map[string]*ast.File) *Workspace {
	return &Workspace{fset, info, pkgNames, files}
}
