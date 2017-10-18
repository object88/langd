package langd

import (
	"go/ast"

	"github.com/object88/langd/collections"
)

// Package is our representation of a Go package
type Package struct {
	// key      collections.Key
	path   string
	astPkg *ast.Package
	// buildPkg *build.Package
	checked bool
}

// Key returns the Key to support the collections.Caravan
func (p *Package) Key() collections.Key {
	return collections.Key(p.path)
}
