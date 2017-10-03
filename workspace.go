package langd

import (
	"fmt"
	"go/ast"
	"go/token"
	"go/types"
	"sync"

	"github.com/object88/rope"
)

// Workspace is a mass of code
type Workspace struct {
	Fset        *token.FileSet
	Info        *types.Info
	PkgNames    map[string]bool
	Files       map[string]*ast.File
	OpenedFiles map[string]*rope.Rope
	rwm         sync.RWMutex
}

// CreateWorkspace returns a new instance of the Workspace struct
func CreateWorkspace() *Workspace {
	openedFiles := map[string]*rope.Rope{}

	return &Workspace{
		OpenedFiles: openedFiles,
	}
}

// AssignAST will inform the workspace of its file set, info, paths, etc.
func (w *Workspace) AssignAST(fset *token.FileSet, info *types.Info, loadedPaths map[string]bool, files map[string]*ast.File) {
	w.Fset = fset
	w.Info = info
	w.PkgNames = loadedPaths
	w.Files = files
}

func (w *Workspace) LocateIdent(p *token.Position) (*ast.Ident, error) {
	if _, ok := w.OpenedFiles[p.Filename]; ok {
		// Force reprocessing the AST before we can continue.
	}

	f := w.Files[p.Filename]
	if f == nil {
		// Failure response is failure.
		return nil, fmt.Errorf("File %s isn't in our workspace\n", p.Filename)
	}

	var x *ast.Ident

	ast.Inspect(f, func(n ast.Node) bool {
		if n == nil {
			return false
		}
		pStart := w.Fset.Position(n.Pos())
		pEnd := w.Fset.Position(n.End())

		if WithinPosition(p, &pStart, &pEnd) {
			switch v := n.(type) {
			case *ast.Ident:
				x = v
			default:
			}
			return true
		}
		return false
	})

	return x, nil
}

// Lock will synchronize access to the workspace for read or write access
func (w *Workspace) Lock(write bool) {
	if write {
		w.rwm.Lock()
	} else {
		w.rwm.RLock()
	}
}

// Unlock will synchronize access to the workspace for read or write access
func (w *Workspace) Unlock(write bool) {
	if write {
		w.rwm.Unlock()
	} else {
		w.rwm.RUnlock()
	}
}
