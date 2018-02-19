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
	rwm sync.RWMutex

	Loader *Loader

	log *log.Log
}

// CreateWorkspace returns a new instance of the Workspace struct
func CreateWorkspace(loader *Loader, log *log.Log) *Workspace {
	return &Workspace{
		Loader: loader,
		log:    log,
	}
}

// ChangeFile applies changes to an opened file
func (w *Workspace) ChangeFile(absFilepath string, startLine, startCharacter, endLine, endCharacter int, text string) error {
	buf, ok := w.Loader.openedFiles[absFilepath]
	if !ok {
		return fmt.Errorf("File %s is not opened", absFilepath)
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

	if err = buf.Alter(startOffset, endOffset, text); err != nil {
		return err
	}

	absPath := filepath.Dir(absFilepath)
	n, ok := w.Loader.caravan.Find(absPath)

	if !ok {
		// Crapola.
		return fmt.Errorf("Failed to find package for file %s", absFilepath)
	}
	p := n.Element.(*Package)

	p.loadState = unloaded
	p.ResetChecker()
	w.Loader.done = false
	w.Loader.stateChange <- absPath

	asc := flattenAscendants(n)

	for _, p1 := range asc {
		p1.loadState = unloaded
		p1.ResetChecker()
		w.Loader.stateChange <- p1.absPath
	}

	return nil
}

// CloseFile will take a file out of the OpenedFiles struct and reparse
func (w *Workspace) CloseFile(absPath string) error {
	_, ok := w.Loader.openedFiles[absPath]
	if !ok {
		w.log.Warnf("File %s is not opened\n", absPath)
		return nil
	}

	delete(w.Loader.openedFiles, absPath)

	w.log.Debugf("File %s is closed\n", absPath)

	return nil
}

// LocateIdent scans the loaded fset for the identifier at the requested
// position
func (w *Workspace) LocateIdent(p *token.Position) (*ast.Ident, error) {
	absPath := filepath.Dir(p.Filename)

	n, ok := w.Loader.caravan.Find(absPath)
	if !ok {
		return nil, fmt.Errorf("No package loaded for '%s'", p.Filename)
	}
	pkg := n.Element.(*Package)
	fi := pkg.files[filepath.Base(p.Filename)]
	f := fi.file

	if f == nil {
		// Failure response is failure.
		return nil, fmt.Errorf("File %s isn't in our workspace", p.Filename)
	}

	var x *ast.Ident

	ast.Inspect(f, func(n ast.Node) bool {
		if n == nil {
			return false
		}
		pStart := pkg.Fset.Position(n.Pos())
		pEnd := pkg.Fset.Position(n.End())

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
	obj, pkg, err := w.locateDeclaration(p)
	if err != nil {
		return nil, err
	}

	declPos := pkg.Fset.Position(obj.Pos())

	return &declPos, nil
}

// LocateReferences returns the array of positions where the given identifier
// is referenced or used
func (w *Workspace) LocateReferences(p *token.Position) []token.Position {
	// Get declaration position, ident, and package
	obj, pkg, err := w.locateDeclaration(p)
	if err != nil {
		// Crappy crap.
		return nil
	}

	// TODO: If declaration should be included in results set, add to `ps`

	refs := w.locateReferences(obj, pkg)

	ps := make([]token.Position, len(refs)+1)
	ps[0] = pkg.Fset.Position(obj.Pos())
	i := 1
	for _, v := range refs {
		ps[i] = v.pkg.Fset.Position(v.pos)
		i++
	}

	return ps
}

// OpenFile shadows the file read from the disk with an in-memory version,
// which the workspace can accept edits to.
func (w *Workspace) OpenFile(absFilepath, text string) error {
	if _, ok := w.Loader.openedFiles[absFilepath]; ok {
		return fmt.Errorf("File %s is already opened", absFilepath)
	}
	w.Loader.openedFiles[absFilepath] = rope.CreateRope(text)

	absPath := filepath.Dir(absFilepath)
	n, ok := w.Loader.caravan.Find(absPath)

	if !ok {
		// Crapola.
		return fmt.Errorf("Failed to find package for file %s", absFilepath)
	}
	p := n.Element.(*Package)

	p.loadState = unloaded
	p.ResetChecker()
	w.Loader.done = false
	w.Loader.stateChange <- absPath

	asc := flattenAscendants(n)

	for _, p1 := range asc {
		p1.loadState = unloaded
		p1.ResetChecker()
		w.Loader.stateChange <- p1.absPath
	}

	w.log.Debugf("Shadowed file '%s'\n", absFilepath)

	return nil
}

// ReplaceFile replaces the entire contents of an opened file
func (w *Workspace) ReplaceFile(absFilepath, text string) error {
	_, ok := w.Loader.openedFiles[absFilepath]
	if !ok {
		return fmt.Errorf("File %s is not opened", absFilepath)
	}

	// Replace the entire document
	buf := rope.CreateRope(text)
	w.Loader.openedFiles[absFilepath] = buf

	absPath := filepath.Dir(absFilepath)
	n, ok := w.Loader.caravan.Find(absPath)

	if !ok {
		// Crapola.
		return fmt.Errorf("Failed to find package for file %s", absFilepath)
	}
	p := n.Element.(*Package)

	p.loadState = unloaded
	p.ResetChecker()
	w.Loader.done = false
	w.Loader.stateChange <- absPath

	asc := flattenAscendants(n)

	for _, p1 := range asc {
		p1.loadState = unloaded
		p1.ResetChecker()
		w.Loader.stateChange <- p1.absPath
	}

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

func (w *Workspace) locateDeclaration(p *token.Position) (types.Object, *Package, error) {
	absPath := filepath.Dir(p.Filename)

	n, ok := w.Loader.caravan.Find(absPath)
	if !ok {
		return nil, nil, fmt.Errorf("No package loaded for '%s'", p.Filename)
	}
	pkg := n.Element.(*Package)
	fi := pkg.files[filepath.Base(p.Filename)]
	f := fi.file

	if f == nil {
		// Failure response is failure.
		return nil, nil, fmt.Errorf("File %s isn't in our workspace", p.Filename)
	}

	var x ast.Node

	fmt.Printf("LocateDeclaration: %s\n", p.String())

	ast.Inspect(f, func(n ast.Node) bool {
		if n == nil {
			return false
		}

		pStart := pkg.Fset.Position(n.Pos())
		pEnd := pkg.Fset.Position(n.End())

		if !WithinPosition(p, &pStart, &pEnd) {
			return false
		}

		switch v := n.(type) {
		case *ast.Ident:
			fmt.Printf("... found ident; %#v\n", v)
			x = v
			return false
		case *ast.SelectorExpr:
			fmt.Printf("... found selector; %#v\n", v)
			selPos := v.Sel
			pSelStart := pkg.Fset.Position(selPos.Pos())
			pSelEnd := pkg.Fset.Position(selPos.End())
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
		return nil, nil, nil
	}

	return w.xyz(x, pkg)
}

func (w *Workspace) xyz(x ast.Node, pkg *Package) (types.Object, *Package, error) {
	switch v := x.(type) {
	case *ast.Ident:
		fmt.Printf("Have ident %#v\n", v)
		if v.Obj != nil {
			fmt.Printf("Ident has obj %#v (%d)\n", v.Obj, v.Pos())
			vObj := pkg.checker.ObjectOf(v)
			return vObj, pkg, nil
		}
		if vDef, ok := pkg.checker.Defs[v]; ok {
			fmt.Printf("Have vDef from Defs: %#v\n", vDef)
			return vDef, pkg, nil
		}
		if vUse, ok := pkg.checker.Uses[v]; ok {
			// Used when var is defined in a package, in another file
			fmt.Printf("Have vUse from Uses: %#v\n", vUse)
			return vUse, pkg, nil
		}

	case *ast.SelectorExpr:
		return w.processSelectorExpr(v, pkg)

	default:
		fmt.Printf("Is %#v\n", x)
	}

	return nil, nil, nil
}

func (w *Workspace) processSelectorExpr(v *ast.SelectorExpr, pkg *Package) (types.Object, *Package, error) {
	fmt.Printf("Have SelectorExpr\n")
	switch vX := v.X.(type) {
	case *ast.Ident:
		vXObj := pkg.checker.ObjectOf(vX)
		if vXObj == nil {
			return nil, nil, fmt.Errorf("v.X (%s) not in ObjectOf", vX.Name)
		}
		fmt.Printf("checker.ObjectOf(v.X): %#v\n", vXObj)
		switch v1 := vXObj.(type) {
		case *types.PkgName:
			fmt.Printf("Have PkgName %s, type %s\n", v1.Name(), v1.Type())
			absPath := v1.Imported().Path()
			n, _ := w.Loader.caravan.Find(absPath)
			pkg1 := n.Element.(*Package)
			fmt.Printf("From pkg %#v\n", pkg1)

			oooo := pkg1.typesPkg.Scope().Lookup(v.Sel.Name)
			if oooo != nil {
				return oooo, pkg1, nil
			}

		case *types.Var:
			fmt.Printf("Have Var %s, type %s\n\tv1: %#v\n\tv1.Sel: %#v\n", v1.Name(), v1.Type(), v1, v.Sel)
			vSelObj := pkg.checker.ObjectOf(v.Sel)
			path := vSelObj.Pkg().Path()
			n, _ := w.Loader.caravan.Find(path)
			pkg1 := n.Element.(*Package)
			return vSelObj, pkg1, nil
		}
	case *ast.SelectorExpr:
		vSelObj := pkg.checker.ObjectOf(v.Sel)
		path := vSelObj.Pkg().Path()
		n, _ := w.Loader.caravan.Find(path)
		pkg1 := n.Element.(*Package)
		return vSelObj, pkg1, nil
	}

	return nil, nil, nil
}
