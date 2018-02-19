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
	fmt.Printf("Starting with %d local refs\n", len(refs))

	if obj.Exported() {
		fmt.Printf("Exported obj\n")
		n, ok := w.Loader.caravan.Find(pkg.absPath)
		if !ok {
			// Should never get here.
			panic("Shit.")
		}
		fmt.Printf("obj: %#v\n", obj)
		asc := flattenAscendants(n)
		fmt.Printf("Have %d ascendants\n", len(asc))
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

	objNode, _ := w.Loader.caravan.Find(obj.Pkg().Path())
	objPkg := objNode.Element.(*Package)
	objPos := objPkg.Fset.Position(obj.Pos())
	fmt.Printf("Looking for '%s':\n\t%#v\n\tin '%s'\n", obj.Name(), obj, objPos.String())

	for _, pkg := range ascendants {
		// fmt.Printf("Checking %s...\n", pkg.absPath)
		if pkg.absPath == "/Users/bropa18/work/src/github.com/gohugoio/hugo/helpers" {
			// fmt.Printf("Imports:\n")
			// for path := range pkg.importPaths {
			// 	fmt.Printf("\t%s\n", path)
			// }
			fmt.Printf("Uses (%d):\n", len(pkg.checker.Uses))
			for k, v := range pkg.checker.Uses {
				if k.Name == "NewWriter" {
					useNode, _ := w.Loader.caravan.Find(v.Pkg().Path())
					usePkg := useNode.Element.(*Package)
					usePos := usePkg.Fset.Position(v.Pos())
					fmt.Printf("\t%#v ->\n\t\t%#v\n\t\tin '%s'\n", k, v, usePos.String())
				}
			}
		}
		for id, use := range pkg.checker.Uses {
			if sameObj(obj, use) {
				fmt.Printf("\tmatch!  %s\n", id.String())
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
			if _, ok := asc[p.absPath]; !ok {
				asc[p.absPath] = p
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
