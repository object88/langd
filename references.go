package langd

import (
	"fmt"
	"go/token"
	"go/types"

	"github.com/object88/langd/collections"
)

type ref struct {
	dp  *DistinctPackage
	pos token.Pos
}

func (w *Workspace) locateReferences(obj types.Object, dp *DistinctPackage) []*ref {
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
	for id, use := range dp.checker.Uses {
		if sameObj(obj, use) {
			refs = append(refs, &ref{
				dp:  dp,
				pos: id.Pos(),
			})
		}
	}

	if obj.Exported() {
		n, ok := w.Loader.caravan.Find(dp.Hash())
		if !ok {
			// Should never get here.
			panic("Shit.")
		}
		asc := flattenAscendants(false, n)
		ascRefs := w.checkAscendants(asc, obj)
		for _, r := range ascRefs {
			refs = append(refs, r)
		}
	}

	fmt.Printf("Returning %d refs\n", len(refs))
	return refs
}

func (w *Workspace) checkAscendants(ascendants map[string]*DistinctPackage, obj types.Object) []*ref {
	refs := []*ref{}

	for _, dp := range ascendants {
		for id, use := range dp.checker.Uses {
			if sameObj(obj, use) {
				refs = append(refs, &ref{
					dp:  dp,
					pos: id.Pos(),
				})
			}
		}
	}
	return refs
}

func flattenAscendants(includeNodes bool, nodes ...*collections.Node) map[string]*DistinctPackage {
	asc := map[string]*DistinctPackage{}

	var f func(n *collections.Node)
	f = func(n *collections.Node) {
		for _, n1 := range n.Ascendants {
			dp := n1.Element.(*DistinctPackage)
			if _, ok := asc[dp.Package.AbsPath]; !ok {
				asc[dp.Package.AbsPath] = dp
				f(n1)
			}
		}
	}

	for _, n := range nodes {
		if includeNodes {
			dp := n.Element.(*DistinctPackage)
			asc[dp.Package.AbsPath] = dp
		}
		f(n)
	}

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
