package langd

import (
	"fmt"
	"go/ast"
	"go/constant"
	"go/token"
	"go/types"
	"path/filepath"
	"strings"

	"github.com/object88/langd/log"
	"github.com/pkg/errors"
	"github.com/spf13/viper"
)

// Workspace is a mass of code
type Workspace struct {
	Loader        Loader
	LoaderContext LoaderContext

	log *log.Log

	settings *viper.Viper
}

// CreateWorkspace returns a new instance of the Workspace struct
func CreateWorkspace(loader Loader, log *log.Log) *Workspace {
	return &Workspace{
		Loader:   loader,
		log:      log,
		settings: viper.New(),
	}
}

// AssignLoaderContext attaches the new loader context to the workspace.  The
// workspace should start to reload the packages.
func (w *Workspace) AssignLoaderContext(lc LoaderContext) {
	w.LoaderContext = lc
	// TODO: reload packages
}

// ChangeFile applies changes to an opened file
func (w *Workspace) ChangeFile(absFilepath string, startLine, startCharacter, endLine, endCharacter int, text string) error {
	buf, err := w.Loader.OpenedFiles().Get(absFilepath)
	if err != nil {
		return err
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
	w.Loader.InvalidatePackage(absPath)

	fmt.Printf("Reload requested\n")
	return nil
}

// CloseFile will take a file out of the OpenedFiles struct and reparse
func (w *Workspace) CloseFile(absPath string) error {
	if err := w.Loader.OpenedFiles().Close(absPath); err != nil {
		w.log.Warnf(err.Error())
	}

	w.log.Debugf("File %s is closed\n", absPath)

	return nil
}

// Hover supplies the hover text for a given position
func (w *Workspace) Hover(p *token.Position) (string, error) {
	obj, dpkg, err := w.locateDeclaration(p)
	if err != nil {
		fmt.Printf("Have err: %s\n", err.Error())
		return "", err
	}

	var s string
	switch t := obj.(type) {
	case *types.Const:
		s = fmt.Sprintf("const %s.%s %s = %s", dpkg.typesPkg.Name(), obj.Name(), getConstType(t), t.Val().String())
	case *types.Func:
		sig := t.Type().(*types.Signature)
		var sb strings.Builder
		sb.WriteString("func ")
		w.makeReceiver(&sb, obj, dpkg, sig)
		w.makeParamList(&sb, sig)
		w.makeReturnList(&sb, sig.Results())
		s = sb.String()
	case *types.TypeName:
		s = w.makeNamed(obj, dpkg)
	case *types.Var:
		s = w.makeNamed(obj, dpkg)
	default:
		if t == nil {
			fmt.Printf("nil obj\n")
		} else {
			fmt.Printf("t: %#v\nt.Type(): %#v\n", t, t.Type())
		}
	}

	hover := "``` go\n" + s + "\n```"
	return hover, nil
}

func (w *Workspace) makeNamed(obj types.Object, dpkg *DistinctPackage) string {
	var s string
	switch t1 := obj.Type().(type) {
	case *types.Basic:
		s = fmt.Sprintf("%s.%s %s", dpkg.typesPkg.Name(), obj.Name(), getBasicType(t1))
	case *types.Named:
		var sb strings.Builder
		fmt.Fprintf(&sb, "type %s.%s struct {", dpkg.typesPkg.Name(), t1.Obj().Name())
		t1u := t1.Underlying()
		t1us := t1u.(*types.Struct)
		if t1us.NumFields() == 0 {
			fmt.Fprintf(&sb, "}")
		} else {
			for k := 0; k < t1us.NumFields(); k++ {
				f := t1us.Field(k)
				fmt.Fprintf(&sb, "\n\t")
				if !f.Anonymous() {
					fmt.Fprintf(&sb, "%s ", f.Name())
				}
				w.getVarType(&sb, f)
			}
			fmt.Fprintf(&sb, "\n}")
		}
		s = sb.String()
	}

	return s
}

func (w *Workspace) makeReceiver(sb *strings.Builder, obj types.Object, dpkg *DistinctPackage, sig *types.Signature) {
	rec := sig.Recv()
	if rec == nil {
		sb.WriteString(dpkg.typesPkg.Name())
		sb.WriteRune('.')
	} else {
		sb.WriteRune('(')
		name := rec.Name()
		if len(name) != 0 {
			sb.WriteString(name)
			sb.WriteRune(' ')
		}
		w.getVarType(sb, rec)
		sb.WriteString(") ")
	}
	sb.WriteString(obj.Name())
}

func (w *Workspace) makeParamList(sb *strings.Builder, sig *types.Signature) {
	sb.WriteRune('(')
	w.makeTupleList(sb, sig.Params(), sig.Variadic())
	sb.WriteRune(')')
}

func (w *Workspace) makeReturnList(sb *strings.Builder, params *types.Tuple) {
	switch params.Len() {
	case 0:
		return
	case 1:
		sb.WriteRune(' ')
		w.makeTupleList(sb, params, false)
	default:
		sb.WriteString(" (")
		w.makeTupleList(sb, params, false)
		sb.WriteRune(')')
	}
}

func (w *Workspace) makeTupleList(sb *strings.Builder, params *types.Tuple, variadic bool) {
	l := params.Len()
	if l == 0 {
		return
	}

	m := l - 1
	if variadic {
		m--
	}

	for k := 0; k < l; k++ {
		if k != 0 {
			sb.WriteString(", ")
		}

		p := params.At(k)
		name := p.Name()
		if len(name) != 0 {
			sb.WriteString(name)
		}

		if k < m && types.Identical(p.Type(), params.At(k+1).Type()) {
			continue
		}

		if len(name) != 0 {
			sb.WriteRune(' ')
		}

		var f func(typ types.Type)
		f = func(typ types.Type) {
			switch t0 := typ.(type) {
			case *types.Pointer:
				sb.WriteRune('*')
				f(t0.Elem())
			case *types.Basic:
				sb.WriteString(t0.Name())
			case *types.Named:
				t0pkg := t0.Obj().Pkg()
				if t0pkg != nil {
					sb.WriteString(t0pkg.Name())
					sb.WriteRune('.')
				}
				sb.WriteString(t0.Obj().Name())
			case *types.Slice:
				if k == l-1 && variadic {
					sb.WriteString("...")
				} else {
					sb.WriteString("[]")
				}
				f(t0.Elem())
			case *types.Signature:
				sb.WriteString("func")
				w.makeParamList(sb, t0)
			default:
				sb.WriteString("unknown")
			}
		}

		f(p.Type())
	}
}

func (w *Workspace) getVarType(sb *strings.Builder, v *types.Var) {
	dhash := w.LoaderContext.GetDistinctHash()

	var f func(typ types.Type)
	f = func(typ types.Type) {
		switch t := typ.(type) {
		case *types.Basic:
			sb.WriteString(getBasicType(t))
		case *types.Named:
			phash := calculateHashFromString(t.Obj().Pkg().Path())
			chash := combineHashes(phash, dhash)
			n, ok := w.Loader.Caravan().Find(chash)
			if !ok {
				sb.WriteString("error")
			}
			dpkg := n.Element.(*DistinctPackage)
			sb.WriteString(dpkg.typesPkg.Name())
			sb.WriteRune('.')
			sb.WriteString(t.Obj().Name())
		case *types.Pointer:
			sb.WriteRune('*')
			f(t.Elem())
		default:
			sb.WriteString("unknown")
		}
	}
	f(v.Type())
}

func getBasicType(o *types.Basic) string {
	var tName string
	if o.Info()&types.IsUntyped == types.IsUntyped {
		switch o.Kind() {
		case types.UntypedBool:
			tName = "bool"
		case types.UntypedComplex:
			tName = "complex"
		case types.UntypedFloat:
			tName = "float"
		case types.UntypedInt:
			tName = "int"
		case types.UntypedNil:
			tName = "nil"
		case types.UntypedRune:
			tName = "rune"
		case types.UntypedString:
			tName = "string"
		}
	} else {
		tName = o.Name()
	}
	return tName
}

func getConstType(o *types.Const) string {
	switch o.Val().Kind() {
	case constant.Bool:
		return "bool"
	case constant.String:
		return "string"
	case constant.Int:
		return "int"
	case constant.Float:
		return "float"
	case constant.Complex:
		return "complex"
	}
	return "(unknown)"
}

// LocateIdent scans the loaded fset for the identifier at the requested
// position
func (w *Workspace) LocateIdent(p *token.Position) (*ast.Ident, error) {
	absPath := filepath.Dir(p.Filename)

	dhash := w.LoaderContext.GetDistinctHash()
	phash := calculateHashFromString(absPath)
	chash := combineHashes(phash, dhash)
	n, ok := w.Loader.Caravan().Find(chash)
	if !ok {
		return nil, fmt.Errorf("No package loaded for '%s'", p.Filename)
	}
	dpkg := n.Element.(*DistinctPackage)
	fi := dpkg.files[filepath.Base(p.Filename)]
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
		pStart := dpkg.Package.Fset.Position(n.Pos())
		pEnd := dpkg.Package.Fset.Position(n.End())

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
	obj, dp, err := w.locateDeclaration(p)
	if err != nil {
		return nil, err
	}

	if obj == nil {
		return nil, nil
	}

	declPos := dp.Package.Fset.Position(obj.Pos())

	return &declPos, nil
}

// LocateReferences returns the array of positions where the given identifier
// is referenced or used
func (w *Workspace) LocateReferences(p *token.Position) []token.Position {
	// Get declaration position, ident, and package
	obj, dp, err := w.locateDeclaration(p)
	if err != nil {
		// Crappy crap.
		fmt.Printf("Received error looking for declaration at %s:\n\t%s\n", p, err)
		return nil
	}

	// TODO: If declaration should be included in results set, add to `ps`

	refs := w.locateReferences(obj, dp)

	ps := make([]token.Position, len(refs)+1)
	ps[0] = dp.Package.Fset.Position(obj.Pos())
	i := 1
	for _, v := range refs {
		ps[i] = v.dp.Package.Fset.Position(v.pos)
		i++
	}

	return ps
}

// OpenFile shadows the file read from the disk with an in-memory version,
// which the workspace can accept edits to.
func (w *Workspace) OpenFile(absFilepath, text string) error {
	hash := calculateHashFromString(text)
	fmt.Printf("Have new hash 0x%x for '%s'\n", hash, absFilepath)

	if err := w.Loader.OpenedFiles().EnsureOpened(absFilepath, text); err != nil {
		return errors.Wrap(err, "From OpenFile")
	}

	absPath := filepath.Dir(absFilepath)
	dp, _ := w.LoaderContext.EnsureDistinctPackage(absPath)
	existingHash := dp.Package.fileHashes[filepath.Base(absFilepath)]
	if existingHash == hash {
		w.log.Debugf("Shadowed file '%s'; unchanged\n", absFilepath)
		return nil
	}

	w.Loader.InvalidatePackage(absPath)

	w.log.Debugf("Shadowed file '%s'\n", absFilepath)

	return nil
}

// ReplaceFile replaces the entire contents of an opened file
func (w *Workspace) ReplaceFile(absFilepath, text string) error {
	if err := w.Loader.OpenedFiles().Replace(absFilepath, text); err != nil {
		return err
	}

	absPath := filepath.Dir(absFilepath)
	w.Loader.InvalidatePackage(absPath)

	return nil
}

func (w *Workspace) locateDeclaration(p *token.Position) (types.Object, *DistinctPackage, error) {
	absPath := filepath.Dir(p.Filename)

	chash := combineHashes(calculateHashFromString(absPath), w.LoaderContext.GetDistinctHash())
	n, ok := w.Loader.Caravan().Find(chash)
	if !ok {
		return nil, nil, fmt.Errorf("No package loaded for '%s'", p.Filename)
	}
	dpkg, _ := n.Element.(*DistinctPackage)
	fi, ok := dpkg.files[filepath.Base(p.Filename)]
	if !ok {
		panic(fmt.Sprintf("Did not find file '%s' in our workspace", p.Filename))
	}
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

		pStart := dpkg.Package.Fset.Position(n.Pos())
		pEnd := dpkg.Package.Fset.Position(n.End())

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
			pSelStart := dpkg.Package.Fset.Position(selPos.Pos())
			pSelEnd := dpkg.Package.Fset.Position(selPos.End())
			if WithinPosition(p, &pSelStart, &pSelEnd) {
				if dpkg.checker == nil {
					panic(fmt.Sprintf("pkg '%s' does not have checker", dpkg.Package.AbsPath))
				}
				s := dpkg.checker.Selections[v]
				fmt.Printf("Selector: %#v\n", s)
				x = v
				return false
			}
		}

		return true
	})

	if x == nil {
		return nil, nil, errors.New("No x found")
	}

	if dpkg == nil {
		fmt.Printf("No package found for x\n")
		return nil, nil, nil
	}

	return w.xyz(x, dpkg)
}

func (w *Workspace) xyz(x ast.Node, dpkg *DistinctPackage) (types.Object, *DistinctPackage, error) {
	switch v := x.(type) {
	case *ast.Ident:
		fmt.Printf("Have ident %#v\n", v)
		if v.Obj != nil {
			fmt.Printf("Ident has obj %#v (%d)\n", v.Obj, v.Pos())
			vObj := dpkg.checker.ObjectOf(v)
			return vObj, dpkg, nil
		}
		if vDef, ok := dpkg.checker.Defs[v]; ok {
			fmt.Printf("Have vDef from Defs: %#v\n", vDef)
			return vDef, dpkg, nil
		}
		if vUse, ok := dpkg.checker.Uses[v]; ok {
			// Used when var is defined in a package, in another file
			fmt.Printf("Have vUse from Uses: %#v\n", vUse)
			return vUse, dpkg, nil
		}

	case *ast.SelectorExpr:
		return w.processSelectorExpr(v, dpkg)

	default:
		fmt.Printf("Is %#v\n", x)
	}

	return nil, nil, nil
}

func (w *Workspace) processSelectorExpr(v *ast.SelectorExpr, dpkg *DistinctPackage) (types.Object, *DistinctPackage, error) {
	fmt.Printf("Have SelectorExpr\n")
	dhash := w.LoaderContext.GetDistinctHash()

	switch vX := v.X.(type) {
	case *ast.Ident:
		vXObj := dpkg.checker.ObjectOf(vX)
		if vXObj == nil {
			return nil, nil, fmt.Errorf("v.X (%s) not in ObjectOf", vX.Name)
		}
		fmt.Printf("checker.ObjectOf(v.X): %#v\n", vXObj)
		switch v1 := vXObj.(type) {
		case *types.PkgName:
			fmt.Printf("Have PkgName %s, type %s\n", v1.Name(), v1.Type())
			absPath := v1.Imported().Path()

			phash := calculateHashFromString(absPath)
			chash := combineHashes(phash, dhash)
			n, _ := w.Loader.Caravan().Find(chash)
			dpkg1 := n.Element.(*DistinctPackage)
			fmt.Printf("From pkg %#v\n", dpkg1)

			oooo := dpkg1.typesPkg.Scope().Lookup(v.Sel.Name)
			if oooo != nil {
				return oooo, dpkg1, nil
			}

		case *types.Var:
			fmt.Printf("Have Var %s, type %s\n\tv1: %#v\n\tv1.Sel: %#v\n", v1.Name(), v1.Type(), v1, v.Sel)
			vSelObj := dpkg.checker.ObjectOf(v.Sel)
			path := vSelObj.Pkg().Path()
			phash := calculateHashFromString(path)
			chash := combineHashes(phash, dhash)
			n, _ := w.Loader.Caravan().Find(chash)
			dpkg1 := n.Element.(*DistinctPackage)
			return vSelObj, dpkg1, nil
		}
	case *ast.SelectorExpr:
		vSelObj := dpkg.checker.ObjectOf(v.Sel)
		path := vSelObj.Pkg().Path()
		phash := calculateHashFromString(path)
		chash := combineHashes(phash, dhash)
		n, _ := w.Loader.Caravan().Find(chash)
		dpkg1 := n.Element.(*DistinctPackage)
		return vSelObj, dpkg1, nil
	}

	return nil, nil, nil
}
