package langd

import (
	"fmt"
	"go/ast"
	"go/token"
	"path/filepath"
	"sync"

	"github.com/object88/langd/collections"
	"github.com/object88/langd/log"
	"github.com/object88/rope"
)

// Workspace is a mass of code
type Workspace struct {
	// Fset *token.FileSet
	// Info        *types.Info
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
	// w.Fset = w.Loader.fset
	// w.Info = w.Loader.info
	w.Files = map[string]*ast.File{}
	w.Loader.caravan.Iter(func(_ string, node *collections.Node) bool {
		pkg := node.Element.(*Package)
		for fname, file := range pkg.files {
			fpath := filepath.Join(pkg.absPath, fname)
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
		pStart := w.Loader.fset.Position(n.Pos())
		pEnd := w.Loader.fset.Position(n.End())

		if WithinPosition(p, &pStart, &pEnd) {
			switch v := n.(type) {
			case *ast.Ident:
				offset := int(v.NamePos) - int(f.Pos())
				fmt.Printf("Found;     (offset %d) %#v\n", offset, n)
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

func (w *Workspace) LocateDeclaration2(p *token.Position) (*token.Position, error) {
	f := w.Files[p.Filename]
	if f == nil {
		// Failure response is failure.
		return nil, fmt.Errorf("File %s isn't in our workspace\n", p.Filename)
	}

	var x ast.Node

	e, _ := w.Loader.caravan.Find(filepath.Dir(p.Filename))
	pkg := e.Element.(*Package)

	ast.Inspect(f, func(n ast.Node) bool {
		if n == nil {
			return false
		}

		pStart := w.Loader.fset.Position(n.Pos())
		pEnd := w.Loader.fset.Position(n.End())

		fmt.Printf("Checking between %s (%d) and %s (%d)...", pStart.String(), pStart.Offset, pEnd.String(), pEnd.Offset)
		if !WithinPosition(p, &pStart, &pEnd) {
			fmt.Printf(" WITHOUT\n")
			return false
		}

		fmt.Printf(" within\n")
		switch v := n.(type) {
		case *ast.Ident:
			fmt.Printf("Found;     %#v\n", n)
			x = v
			return false
		case *ast.SelectorExpr:
			selPos := v.Sel
			pSelStart := w.Loader.fset.Position(selPos.Pos())
			pSelEnd := w.Loader.fset.Position(selPos.End())
			if WithinPosition(p, &pSelStart, &pSelEnd) {
				s := pkg.info.Selections[v]
				fmt.Printf("Selector: %#v\n", s)
				x = v
				return false
			}
		}

		return true
	})

	if x == nil {
		fmt.Printf("No x found\n")
		return nil, nil
	}

	switch v := x.(type) {
	case *ast.Ident:
		fmt.Printf("Have ident\n")
		if v.Obj != nil {
			fmt.Printf("Ident has obj %#v (%d)\n", v.Obj, v.Pos())
			identPosition := w.Loader.fset.Position(v.Obj.Pos())
			return &identPosition, nil
		}
		// xObj := pkg.info.ObjectOf(v)
		// if xObj != nil {
		// 	identPosition := w.Loader.fset.Position(xObj.Pos())
		// 	return &identPosition, nil
		// }
	case *ast.SelectorExpr:
		fmt.Printf("Have SelectorExpr\n")
		fmt.Printf("X: %#v\n", v.X)
		fmt.Printf("Sel: %#v\n", v.Sel)
		fmt.Printf("Defs:\n")
		for k, v := range pkg.checker.Defs {
			fmt.Printf("\t%#v -> %#v\n", k, v)
		}
		fmt.Printf("Selections:\n")
		for k, v := range pkg.checker.Selections {
			fmt.Printf("\t%#v -> %#v\n", k, v)
		}
		fmt.Printf("Implicits:\n")
		for k, v := range pkg.checker.Implicits {
			fmt.Printf("\t%#v -> %#v\n", k, v)
		}
		fmt.Printf("Uses:\n")
		for k, v := range pkg.checker.Uses {
			fmt.Printf("\t%#v -> %#v\n", k, v)
		}
		vXObj := pkg.checker.ObjectOf(v.X.(*ast.Ident))
		if vXObj == nil {
			fmt.Printf("v.X not in ObjectOf\n")
		} else {
			fmt.Printf("checker.ObjectOf(v.X): %#v\n", vXObj)
		}

	default:
		fmt.Printf("Is %#v\n", x)
	}
	return nil, nil
}

func (w *Workspace) expressionType(n ast.Node) {
	switch x := n.(type) {
	case *ast.Ident:
		obj := x.Obj
		fmt.Printf("expressionType: %#v\n", obj)
	}
}

// LocateDeclaration returns the position where the provided identifier is
// declared & defined
func (w *Workspace) LocateDeclaration(x *ast.Ident) *token.Position {
	var xObjPos token.Pos

	fmt.Printf("Name position: %d, %s\n", x.NamePos, w.Loader.fset.Position(x.NamePos).String())
	if x.Obj != nil {
		fmt.Printf("Obj position: %d, %s\n", x.Obj.Pos(), w.Loader.fset.Position(x.Obj.Pos()).String())
	}

	// xObj := x.Obj
	// if xObj != nil {
	// 	xObjPos = xObj.Pos()
	// } else {
	// 	xObj := w.Info.ObjectOf(x)
	// 	if xObj != nil {
	// 		xObjPos = xObj.Pos()
	// 	} else {
	// 		xObj := w.Info.Defs[x]
	// 		if xObj != nil {
	// 			xObjPos = xObj.Pos()
	// 		} else {
	// 			fmt.Printf("Ident did not have Obj and not in Info.ObjectOf and not in Info.Defs\n")
	// 			return nil
	// 		}
	// 	}
	// }
	fmt.Printf("Got xObjPos: %d\n", xObjPos)
	loc := w.Loader.fset.Position(xObjPos)
	fmt.Printf("Got loc:     %s\n", loc.String())
	return &loc
}

// LocateReferences returns the array of positions where the given identifier
// is referenced or used
func (w *Workspace) LocateReferences(x *ast.Ident) *[]token.Position {
	// xObj := w.Info.ObjectOf(x)
	ps := []token.Position{}

	// for k, v := range w.Info.Uses {
	// 	if xObj == v {
	// 		ps = append(ps, w.Fset.Position(k.Pos()))
	// 	}
	// }

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
