package langd

import (
	"fmt"
	"go/ast"
	"go/parser"
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

// ChangeFile applies changes to an opened file
func (w *Workspace) ChangeFile(absPath string, startLine, startCharacter, endLine, endCharacter int, text string) error {
	buf, ok := w.OpenedFiles[absPath]
	if !ok {
		return fmt.Errorf("File %s is not opened\n", absPath)
	}

	// Have position (line, character), need to transform into offset into file
	// Then replace starting from there.
	r1 := buf.NewReader()
	startOffset, err := CalculateOffsetForPosition(r1, startLine, startCharacter)
	if err != nil {
		// Crap crap crap crap.
		fmt.Printf("Error from start: %s", err.Error())
	}

	r2 := buf.NewReader()
	endOffset, err := CalculateOffsetForPosition(r2, endLine, endCharacter)
	if err != nil {
		// Crap crap crap crap.
		fmt.Printf("Error from end: %s", err.Error())
	}

	fmt.Printf("offsets: [%d:%d]\n", startOffset, endOffset)

	buf.Alter(startOffset, endOffset, text)

	return nil
}

// CloseFile will take a file out of the OpenedFiles struct and reparse
func (w *Workspace) CloseFile(absPath string) error {
	_, ok := w.OpenedFiles[absPath]
	if !ok {
		w.log.Warnf("File %s is not opened\n", absPath)
		return nil
	}

	w.log.Debugf("File %s is open...\n", absPath)
	delete(w.OpenedFiles, absPath)

	astFile, err := parser.ParseFile(w.Loader.Fset, absPath, nil, 0)
	if err != nil {
		w.log.Errorf("Failed to parse file as provided by didOpen: %s\n", err.Error())
	}

	w.Files[absPath] = astFile

	w.log.Debugf("File %s is closed\n", absPath)

	return nil
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
		pStart := w.Loader.Fset.Position(n.Pos())
		pEnd := w.Loader.Fset.Position(n.End())

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

// LocateDeclaration returns the position where the provided identifier is
// declared & defined
func (w *Workspace) LocateDeclaration(p *token.Position) (*token.Position, error) {
	f := w.Files[p.Filename]
	if f == nil {
		// Failure response is failure.
		return nil, fmt.Errorf("File %s isn't in our workspace\n", p.Filename)
	}

	var x ast.Node

	fmt.Printf("LocateDeclaration: %s\n", p.Filename)

	e, _ := w.Loader.caravan.Find(filepath.Dir(p.Filename))
	pkg := e.Element.(*Package)

	ast.Inspect(f, func(n ast.Node) bool {
		if n == nil {
			return false
		}

		pStart := w.Loader.Fset.Position(n.Pos())
		pEnd := w.Loader.Fset.Position(n.End())

		if !WithinPosition(p, &pStart, &pEnd) {
			return false
		}

		switch v := n.(type) {
		case *ast.Ident:
			fmt.Printf("Found;     %#v\n", n)
			x = v
			return false
		case *ast.SelectorExpr:
			selPos := v.Sel
			pSelStart := w.Loader.Fset.Position(selPos.Pos())
			pSelEnd := w.Loader.Fset.Position(selPos.End())
			if WithinPosition(p, &pSelStart, &pSelEnd) {
				s := pkg.checker.Selections[v]
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
			identPosition := w.Loader.Fset.Position(v.Obj.Pos())
			return &identPosition, nil
		}
		// xObj := pkg.info.ObjectOf(v)
		// if xObj != nil {
		// 	identPosition := w.Loader.Fset.Position(xObj.Pos())
		// 	return &identPosition, nil
		// }
	case *ast.SelectorExpr:
		fmt.Printf("Have SelectorExpr\n")

		scopedObj := f.Scope.Lookup(v.X.(*ast.Ident).Name)
		fmt.Printf("Scoped object... %#v\n", scopedObj)

		vXObj := pkg.checker.ObjectOf(v.X.(*ast.Ident))
		if vXObj == nil {
			fmt.Printf("v.X not in ObjectOf\n")
		} else {
			fmt.Printf("checker.ObjectOf(v.X): %#v\n", vXObj)
			switch v1 := vXObj.(type) {
			case *types.PkgName:
				fmt.Printf("Have PkgName %s, type %s\n", v1.Name(), v1.Type())
				absPath := v1.Imported().Path()
				e, _ := w.Loader.caravan.Find(absPath)
				pkg1 := e.Element.(*Package)
				fmt.Printf("From pkg %#v\n", pkg1)

				oooo := pkg1.typesPkg.Scope().Lookup(v.Sel.Name)
				if oooo != nil {
					// Have thingy from scope!
					declPos := w.Loader.Fset.Position(oooo.Pos())
					return &declPos, nil
				}

				selIdent := ast.NewIdent(v.Sel.Name)
				fmt.Printf("Using new ident %#v\n", selIdent)
				def, ok := pkg1.checker.Defs[selIdent]
				if !ok {
					fmt.Printf("Not from Defs\n")
				} else {
					fmt.Printf("From defs: %#v\n", def)
					declPos := w.Loader.Fset.Position(def.Pos())
					return &declPos, nil
				}
			}
		}

	default:
		fmt.Printf("Is %#v\n", x)
	}
	return nil, nil
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

// OpenFile shadows the file read from the disk with an in-memory version,
// which the workspace can accept edits to.
func (w *Workspace) OpenFile(absPath, text string) error {
	w.OpenedFiles[absPath] = rope.CreateRope(text)

	// DISABLE UNTIL WE ARE ABLE TO RERUN TYPECHECKER
	// astFile, err := parser.ParseFile(w.Loader.Fset, rh.fpath, rh.text, 0)
	// if err != nil {
	// 	rh.h.log.Warnf("Failed to parse file as provided by didOpen: %s\n", err.Error())
	// }

	// w.Files[rh.fpath] = astFile

	w.log.Debugf("Shadowed file '%s'\n", absPath)

	return nil
}

// ReplaceFile replaces the entire contents of an opened file
func (w *Workspace) ReplaceFile(absPath, text string) error {
	_, ok := w.OpenedFiles[absPath]
	if !ok {
		return fmt.Errorf("File %s is not opened\n", absPath)
	}

	// Replace the entire document
	buf := rope.CreateRope(text)
	w.OpenedFiles[absPath] = buf

	return nil
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
