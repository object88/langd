package langd

import (
	"go/ast"
	"go/types"
)

// Package represents a Go package
type Package struct {
	AstPkg      *ast.Package
	AstPackages map[string]*ast.Package
	Info        *types.Info
	Name        string
	Pkg         *types.Package
}

func newPkg(name string) *Package {
	info := &types.Info{
		Types: make(map[ast.Expr]types.TypeAndValue),
		Defs:  make(map[*ast.Ident]types.Object),
		Uses:  make(map[*ast.Ident]types.Object),
	}

	return &Package{nil, nil, info, name, nil}
}
