package langd

import (
	"fmt"
	"go/ast"
	"go/token"
	"go/types"
	"path/filepath"
	"sync"

	"github.com/object88/langd/collections"
	"github.com/object88/langd/log"
	"github.com/object88/rope"
)

// Workspace is a mass of code
type Workspace struct {
	Fset        *token.FileSet
	Info        *types.Info
	Files       map[string]*ast.File
	OpenedFiles map[string]*rope.Rope
	rwm         sync.RWMutex

	Loader *Loader

	log *log.Log
}

// CreateWorkspace returns a new instance of the Workspace struct
func CreateWorkspace(loader *Loader, log *log.Log) *Workspace {
	openedFiles := map[string]*rope.Rope{}

	return &Workspace{
		OpenedFiles: openedFiles,
		Loader:      loader,
		log:         log,
	}
}

// AssignAST will inform the workspace of its file set, info, paths, etc.
func (w *Workspace) AssignAST() {
	w.Fset = w.Loader.fset
	w.Info = w.Loader.info
	w.Files = map[string]*ast.File{}
	w.Loader.caravan.Iter(func(_ collections.Key, node *collections.Node) bool {
		pkg := node.Element.(*Package)
		for fname, file := range pkg.files {
			fpath := filepath.Join(pkg.absPath.String(), fname)
			w.Files[fpath] = file.file
		}
		return true
	})
}

// LocateIdent scans the loaded fset for the identifier at the requested
// position
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
				fmt.Printf("Found;     %#v\n", n)
				x = v
				return false
			default:
				fmt.Printf("Narrowing; %#v\n", n)
			}
			return true
		}
		return false
	})

	return x, nil
}

// LocateDeclaration returns the position where the provided identifier is
// declared & defined
func (w *Workspace) LocateDeclaration(x *ast.Ident) *token.Position {
	xObj := w.Info.ObjectOf(x)
	fmt.Printf("Got xObj:    %#v\n", xObj)
	if xObj == nil {
		return nil
	}
	xObjPos := xObj.Pos()
	fmt.Printf("Got xObjPos: %d\n", xObjPos)
	loc := w.Fset.Position(xObjPos)
	fmt.Printf("Got loc:     %s\n", loc.String())
	return &loc
}

// LocateReferences returns the array of positions where the given identifier
// is referenced or used
func (w *Workspace) LocateReferences(x *ast.Ident) *[]token.Position {
	xObj := w.Info.ObjectOf(x)
	ps := []token.Position{}

	for k, v := range w.Info.Uses {
		if xObj == v {
			ps = append(ps, w.Fset.Position(k.Pos()))
		}
	}

	return &ps
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
