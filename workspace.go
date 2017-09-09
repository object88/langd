package langd

import (
	"go/token"
)

// Workspace is a mass of code
type Workspace struct {
	Fset *token.FileSet
	Pkgs map[string]*Package
}

func newWorkspace(fset *token.FileSet, pkgs map[string]*Package) *Workspace {
	return &Workspace{fset, pkgs}
}
