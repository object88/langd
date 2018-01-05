package langd

import (
	"fmt"
	"go/ast"
	"go/token"
	"go/types"
	"path/filepath"
	"sync"

	"github.com/object88/langd/log"
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
	w.PkgNames = make(map[string]bool, len(w.Loader.directories))
	w.Files = map[string]*ast.File{}
	for _, v := range w.Loader.directories {
		for k, pkg := range v.pm {
			w.PkgNames[k] = true
			for fname, astf := range pkg.files {
				fpath := filepath.Join(v.absPath, fname)
				w.Files[fpath] = astf
			}
		}
	}
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
			case *ast.AssignStmt:
				x = v.Lhs[0].(*ast.Ident)
			// case *ast.CallExpr:
			// 	fmt.Printf("SOMETHING SOMETHING\n\n%#v\n\n", v)
			// 	fIndent := v.Fun.(*ast.Ident)
			// 	fStart := w.Fset.Position(fIndent.Pos())
			// 	fEnd := w.Fset.Position(fIndent.End())
			// 	if WithinPosition(p, &fStart, &fEnd) {
			// 		fmt.Printf("It is the func!\n")
			// 	} else {
			// 		fmt.Printf("Func args?\n")
			// 		ids := v.Args
			// 		x = ids[0].(*ast.Ident)
			// 	}
			case *ast.FuncDecl:
				x = v.Name
			case *ast.Ident:
				x = v
			default:
				fmt.Printf("Narrowing; %#v\n", n)
			}
			return true
		}
		return false
	})

	return x, nil
}

func (w *Workspace) LocateDefinition(x *ast.Ident) *token.Position {
	if x.Obj == nil {
		p := w.Fset.Position(x.NamePos)
		return &p
	}

	var declPosition token.Position
	switch v1 := x.Obj.Decl.(type) {
	case *ast.AssignStmt:
		declPosition = w.Fset.Position(v1.Pos())
		w.log.Verbosef("Have assign; declaration at %s\n", declPosition.String())

	case *ast.Field:
		declPosition = w.Fset.Position(v1.Pos())
		w.log.Verbosef("Have field; declaration at %s\n", declPosition.String())

	case *ast.FuncDecl:
		declPosition = w.Fset.Position(v1.Name.Pos())
		w.log.Verbosef("Have func; declaration at %s\n", declPosition.String())

	case *ast.TypeSpec:
		declPosition = w.Fset.Position(v1.Pos())
		w.log.Verbosef("Have typespec; declaration at %s\n", declPosition.String())

	case *ast.ValueSpec:
		declPosition = w.Fset.Position(v1.Pos())
		w.log.Verbosef("Have valuespec; declaration at %s\n", declPosition.String())

	default:
		// No-op
		w.log.Verbosef("Have identifier: %s, object %#v\n", x.String(), x.Obj.Decl)
		return nil
	}

	return &declPosition
}

func (w *Workspace) LocateReferences(x *ast.Ident) *[]token.Position {
	obj := w.Info.ObjectOf(x)
	ps := []token.Position{}

	for k, v := range w.Info.Uses {
		if obj == v {
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
