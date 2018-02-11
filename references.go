package langd

import (
	"go/token"
	"go/types"
)

type ref struct {
	pkg *Package
	pos token.Pos
}

// func (w *Workspace) locateReferences(ident *ast.Ident, pkg *Package) []*ref {
func (w *Workspace) locateReferences(obj types.Object, pkg *Package) []*ref {
	// obj := pkg.checker.ObjectOf(ident)
	if obj == nil {
		// Special case
	}

	// If this is a package name, do something special also.
	if _, ok := obj.(*types.PkgName); ok {
		// *shrug*
	}

	if obj.Pkg() == nil {
		// Uhhh, not sure what this is?  A keyword or something?
	}

	// Start off with in-package references, shall we?
	var refs []*ref
	for id, use := range pkg.checker.Uses {
		if sameObj(obj, use) {
			refs = append(refs, &ref{
				pkg: pkg,
				pos: id.Pos(),
			})
		}
	}
	return refs
}

func sameObj(x, y types.Object) bool {
	if x == y {
		return true
	}

	xPkgname, ok := x.(*types.PkgName)
	if !ok {
		return false
	}
	yPkgname, ok := y.(*types.PkgName)
	if !ok {
		return false
	}

	return xPkgname.Imported() == yPkgname.Imported()
}
