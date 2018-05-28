package langd

import (
	"fmt"
	"go/token"
	"go/types"

	"github.com/object88/langd/collections"
)

type ref struct {
	pkg *Package
	pos token.Pos
}

// func (w *Workspace) locateReferences(ident *ast.Ident, pkg *Package) []*ref {
func (w *Workspace) locateReferences(obj types.Object, pkg *Package) []*ref {
	// obj := pkg.checker.ObjectOf(ident)
	if obj == nil {
		fmt.Printf("No obj provided\n")
		// Special case
	}

	// If this is a package name, do something special also.
	if _, ok := obj.(*types.PkgName); ok {
		// *shrug*
		fmt.Printf("Package name\n")
	}

	if obj.Pkg() == nil {
		// Uhhh, not sure what this is?  A keyword or something?
		fmt.Printf("obj.Pkg is nil\n")
	}

	dp := pkg.distincts[w.LoaderContext.GetDistinctHash()]

	// Start off with in-package references, shall we?
	var refs []*ref
	for id, use := range dp.checker.Uses {
		if sameObj(obj, use) {
			refs = append(refs, &ref{
				pkg: pkg,
				pos: id.Pos(),
			})
		}
	}

	if obj.Exported() {
		hash := BuildPackageHash(pkg.AbsPath)
		n, ok := w.Loader.Caravan().Find(hash)
		if !ok {
			// Should never get here.
			panic("Shit.")
		}
		asc := flattenAscendants(n)
		ascRefs := w.checkAscendants(asc, obj)
		for _, r := range ascRefs {
			refs = append(refs, r)
		}
	}

	fmt.Printf("Returning %d refs\n", len(refs))
	return refs
}

func (w *Workspace) checkAscendants(ascendants map[string]*Package, obj types.Object) []*ref {
	refs := []*ref{}

	hash := w.LoaderContext.GetDistinctHash()
	for _, pkg := range ascendants {
		dpkg := pkg.distincts[hash]
		for id, use := range dpkg.checker.Uses {
			if sameObj(obj, use) {
				refs = append(refs, &ref{
					pkg: pkg,
					pos: id.Pos(),
				})
			}
		}
	}
	return refs
}

func flattenAscendants(n *collections.Node) map[string]*Package {
	asc := map[string]*Package{}

	var f func(n *collections.Node)
	f = func(n *collections.Node) {
		for _, n1 := range n.Ascendants {
			p := n1.Element.(*Package)
			if _, ok := asc[p.AbsPath]; !ok {
				asc[p.AbsPath] = p
				f(n1)
			}
		}
	}

	f(n)

	return asc
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
