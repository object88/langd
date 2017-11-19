package langd

import (
	"fmt"
	"go/ast"

	"github.com/object88/langd/collections"
)

// Package is our representation of a Go package
type Package struct {
	path    string
	pkgName string
	astPkg  *ast.Package
	checked bool
}

// Key returns the Key to support the collections.Caravan.  The Key is a
// colon-seperated concatination of the package name and the path.
func (p *Package) Key() collections.Key {
	key := buildKey(p.pkgName, p.path)
	return key
}

func buildKey(pkgName, path string) collections.Key {
	k := fmt.Sprintf("%s:%s", pkgName, path)
	return collections.Key(k)
}
