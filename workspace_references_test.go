package langd

import (
	"go/token"
	"io/ioutil"
	"path/filepath"
	"strings"
	"testing"
)

func Test_Workspace_References_Local_Const(t *testing.T) {
	src1 := `package foo
	const fooVal = 0
	func IncFoo() int {
		return fooVal
	}`

	rootPath, overlayFs := createOverlay(map[string]string{
		filepath.Join("foo", "foo.go"): src1,
	})
	fooPath := filepath.Join(rootPath, "foo")
	fooGoPath := filepath.Join(rootPath, "foo", "foo.go")

	w, closer := workspaceSetup(t, fooPath, overlayFs, false)
	defer closer()

	startPosition := &token.Position{Filename: fooGoPath, Line: 4, Column: 10}
	referencePositions := []*token.Position{
		&token.Position{Filename: fooGoPath, Line: 2, Column: 8},
		startPosition,
	}
	testReferences(t, w, startPosition, referencePositions)
}

func Test_Workspace_References_Package_Const(t *testing.T) {
	src1 := `package foo
	const fooVal = 0`

	src2 := `package foo
	func IncFoo() int {
		return fooVal
	}`

	rootPath, overlayFs := createOverlay(map[string]string{
		filepath.Join("foo", "foo1.go"): src1,
		filepath.Join("foo", "foo2.go"): src2,
	})
	fooPath := filepath.Join(rootPath, "foo")
	foo1GoPath := filepath.Join(rootPath, "foo", "foo1.go")
	foo2GoPath := filepath.Join(rootPath, "foo", "foo2.go")

	w, closer := workspaceSetup(t, fooPath, overlayFs, false)
	defer closer()

	startPosition := &token.Position{Filename: foo2GoPath, Line: 3, Column: 10}
	referencePositions := []*token.Position{
		&token.Position{Filename: foo1GoPath, Line: 2, Column: 8},
		startPosition,
	}
	testReferences(t, w, startPosition, referencePositions)
}

func Test_Workspace_References_Imported_Const(t *testing.T) {
	src1 := `package foo
	const FooVal = 0`

	src2 := `package bar
	import "../foo"
	func IncFoo() int {
		return foo.FooVal
	}`

	rootPath, overlayFs := createOverlay(map[string]string{
		filepath.Join("foo", "foo.go"): src1,
		filepath.Join("bar", "bar.go"): src2,
	})
	barPath := filepath.Join(rootPath, "bar")
	fooGoPath := filepath.Join(rootPath, "foo", "foo.go")
	barGoPath := filepath.Join(rootPath, "bar", "bar.go")

	w, closer := workspaceSetup(t, barPath, overlayFs, false)
	defer closer()

	startPosition := &token.Position{Filename: barGoPath, Line: 4, Column: 14}
	referencePositions := []*token.Position{
		&token.Position{Filename: fooGoPath, Line: 2, Column: 8},
		startPosition,
	}
	testReferences(t, w, startPosition, referencePositions)
}

func Test_Workspace_References_Local_Var(t *testing.T) {
	src1 := `package foo
	var fooVal int = 0
	func IncFoo() int {
		fooVal++
		return fooVal
	}`

	rootPath, overlayFs := createOverlay(map[string]string{
		filepath.Join("foo", "foo.go"): src1,
	})
	fooPath := filepath.Join(rootPath, "foo")
	fooGoPath := filepath.Join(rootPath, "foo", "foo.go")

	w, closer := workspaceSetup(t, fooPath, overlayFs, false)
	defer closer()

	startPosition := &token.Position{Filename: fooGoPath, Line: 4, Column: 3}
	referencePositions := []*token.Position{
		&token.Position{Filename: fooGoPath, Line: 2, Column: 6},
		startPosition,
		&token.Position{Filename: fooGoPath, Line: 5, Column: 10},
	}
	testReferences(t, w, startPosition, referencePositions)
}

func Test_Workspace_References_Package_Var(t *testing.T) {
	src1 := `package foo
	var fooVal int = 0`

	src2 := `package foo
	func IncFoo() int {
		fooVal++
		return fooVal
	}`

	rootPath, overlayFs := createOverlay(map[string]string{
		filepath.Join("foo", "foo1.go"): src1,
		filepath.Join("foo", "foo2.go"): src2,
	})
	fooPath := filepath.Join(rootPath, "foo")
	foo1GoPath := filepath.Join(rootPath, "foo", "foo1.go")
	foo2GoPath := filepath.Join(rootPath, "foo", "foo2.go")

	w, closer := workspaceSetup(t, fooPath, overlayFs, false)
	defer closer()

	startPosition := &token.Position{Filename: foo2GoPath, Line: 3, Column: 3}
	referencePositions := []*token.Position{
		&token.Position{Filename: foo1GoPath, Line: 2, Column: 6},
		startPosition,
		&token.Position{Filename: foo2GoPath, Line: 4, Column: 10},
	}
	testReferences(t, w, startPosition, referencePositions)
}

func Test_Workspace_References_Imported_Var(t *testing.T) {
	src1 := `package foo
	var FooVal int = 0`

	src2 := `package bar
	import "../foo"
	func IncFoo() int {
		return foo.FooVal
	}`

	rootPath, overlayFs := createOverlay(map[string]string{
		filepath.Join("foo", "foo.go"): src1,
		filepath.Join("bar", "bar.go"): src2,
	})
	barPath := filepath.Join(rootPath, "bar")
	fooGoPath := filepath.Join(rootPath, "foo", "foo.go")
	barGoPath := filepath.Join(rootPath, "bar", "bar.go")

	w, closer := workspaceSetup(t, barPath, overlayFs, false)
	defer closer()

	startPosition := &token.Position{Filename: barGoPath, Line: 4, Column: 14}
	referencePositions := []*token.Position{
		&token.Position{Filename: fooGoPath, Line: 2, Column: 6},
		startPosition,
	}
	testReferences(t, w, startPosition, referencePositions)
}

func Test_Workspace_References_Local_Struct(t *testing.T) {
	src1 := `package foo
	type fooStruct struct {
		a string
	}
	func Do() {
		f := fooStruct{}
		f.a = "astring"
	}`

	rootPath, overlayFs := createOverlay(map[string]string{
		filepath.Join("foo", "foo.go"): src1,
	})
	fooPath := filepath.Join(rootPath, "foo")
	fooGoPath := filepath.Join(rootPath, "foo", "foo.go")

	w, closer := workspaceSetup(t, fooPath, overlayFs, false)
	defer closer()

	startPosition := &token.Position{Filename: fooGoPath, Line: 6, Column: 8}
	referencePositions := []*token.Position{
		&token.Position{Filename: fooGoPath, Line: 2, Column: 7},
		startPosition,
	}
	testReferences(t, w, startPosition, referencePositions)
}

func Test_Workspace_References_Package_Struct(t *testing.T) {
	src1 := `package foo
	type fooStruct struct {
		a string
	}`

	src2 := `package foo
	func Do() {
		f := fooStruct{}
		f.a = "astring"
	}`

	rootPath, overlayFs := createOverlay(map[string]string{
		filepath.Join("foo", "foo1.go"): src1,
		filepath.Join("foo", "foo2.go"): src2,
	})
	fooPath := filepath.Join(rootPath, "foo")
	foo1GoPath := filepath.Join(rootPath, "foo", "foo1.go")
	foo2GoPath := filepath.Join(rootPath, "foo", "foo2.go")

	w, closer := workspaceSetup(t, fooPath, overlayFs, false)
	defer closer()

	startPosition := &token.Position{Filename: foo2GoPath, Line: 3, Column: 8}
	referencePositions := []*token.Position{
		&token.Position{Filename: foo1GoPath, Line: 2, Column: 7},
		startPosition,
	}
	testReferences(t, w, startPosition, referencePositions)
}

func Test_Workspace_References_Imported_Struct(t *testing.T) {
	src1 := `package foo
	type FooStruct struct {
		A string
	}`

	src2 := `package bar
	import "../foo"
	func Do() {
		f := foo.FooStruct{}
		f.A = "astring"
	}`

	rootPath, overlayFs := createOverlay(map[string]string{
		filepath.Join("foo", "foo.go"): src1,
		filepath.Join("bar", "bar.go"): src2,
	})
	barPath := filepath.Join(rootPath, "bar")
	fooGoPath := filepath.Join(rootPath, "foo", "foo.go")
	barGoPath := filepath.Join(rootPath, "bar", "bar.go")

	w, closer := workspaceSetup(t, barPath, overlayFs, false)
	defer closer()

	startPosition := &token.Position{Filename: barGoPath, Line: 4, Column: 12}
	referencePositions := []*token.Position{
		&token.Position{Filename: fooGoPath, Line: 2, Column: 7},
		startPosition,
	}
	testReferences(t, w, startPosition, referencePositions)
}

func Test_Workspace_References_Local_Interface(t *testing.T) {
	src1 := `package foo
	type fooIface interface {
		myNumber() int
	}
	func Do() int {
		var f fooIface
		return f.myNumber()
	}`

	rootPath, overlayFs := createOverlay(map[string]string{
		filepath.Join("foo", "foo.go"): src1,
	})
	fooPath := filepath.Join(rootPath, "foo")
	fooGoPath := filepath.Join(rootPath, "foo", "foo.go")

	w, closer := workspaceSetup(t, fooPath, overlayFs, false)
	defer closer()

	startPosition := &token.Position{Filename: fooGoPath, Line: 6, Column: 9}
	referencePositions := []*token.Position{
		&token.Position{Filename: fooGoPath, Line: 2, Column: 7},
		startPosition,
	}
	testReferences(t, w, startPosition, referencePositions)
}

func Test_Workspace_References_Package_Interface(t *testing.T) {
	src1 := `package foo
	type fooIface interface {
		myNumber() int
	}`
	src2 := `package foo
	func Do() int {
		var f fooIface
		return f.myNumber()
	}`

	rootPath, overlayFs := createOverlay(map[string]string{
		filepath.Join("foo", "foo1.go"): src1,
		filepath.Join("foo", "foo2.go"): src2,
	})
	fooPath := filepath.Join(rootPath, "foo")
	foo1GoPath := filepath.Join(rootPath, "foo", "foo1.go")
	foo2GoPath := filepath.Join(rootPath, "foo", "foo2.go")

	w, closer := workspaceSetup(t, fooPath, overlayFs, false)
	defer closer()

	startPosition := &token.Position{Filename: foo2GoPath, Line: 3, Column: 9}
	referencePositions := []*token.Position{
		&token.Position{Filename: foo1GoPath, Line: 2, Column: 7},
		startPosition,
	}
	testReferences(t, w, startPosition, referencePositions)
}

func Test_Workspace_References_Imported_Interface(t *testing.T) {
	src1 := `package foo
	type FooIface interface {
		MyNumber() int
	}`
	src2 := `package bar
	import "../foo"
	func Do() int {
		var f foo.FooIface
		return f.MyNumber()
	}`

	rootPath, overlayFs := createOverlay(map[string]string{
		filepath.Join("foo", "foo.go"): src1,
		filepath.Join("bar", "bar.go"): src2,
	})
	barPath := filepath.Join(rootPath, "bar")
	fooGoPath := filepath.Join(rootPath, "foo", "foo.go")
	barGoPath := filepath.Join(rootPath, "bar", "bar.go")

	w, closer := workspaceSetup(t, barPath, overlayFs, false)
	defer closer()

	startPosition := &token.Position{Filename: barGoPath, Line: 4, Column: 13}
	referencePositions := []*token.Position{
		&token.Position{Filename: fooGoPath, Line: 2, Column: 7},
		startPosition,
	}
	testReferences(t, w, startPosition, referencePositions)
}

func Test_Workspace_References_Local_Func(t *testing.T) {
	src1 := `package foo
	func getFoo() int {
		return 0
	}
	func Do() int {
		return getFoo()
	}`

	rootPath, overlayFs := createOverlay(map[string]string{
		filepath.Join("foo", "foo.go"): src1,
	})
	fooPath := filepath.Join(rootPath, "foo")
	fooGoPath := filepath.Join(rootPath, "foo", "foo.go")

	w, closer := workspaceSetup(t, fooPath, overlayFs, false)
	defer closer()

	startPosition := &token.Position{Filename: fooGoPath, Line: 6, Column: 10}
	referencePositions := []*token.Position{
		&token.Position{Filename: fooGoPath, Line: 2, Column: 7},
		startPosition,
	}
	testReferences(t, w, startPosition, referencePositions)
}

func Test_Workspace_References_Package_Func(t *testing.T) {
	src1 := `package foo
	func getFoo() int {
		return 0
	}`

	src2 := `package foo
	func Do() int {
		return getFoo()
	}`

	rootPath, overlayFs := createOverlay(map[string]string{
		filepath.Join("foo", "foo1.go"): src1,
		filepath.Join("foo", "foo2.go"): src2,
	})
	fooPath := filepath.Join(rootPath, "foo")
	foo1GoPath := filepath.Join(rootPath, "foo", "foo1.go")
	foo2GoPath := filepath.Join(rootPath, "foo", "foo2.go")

	w, closer := workspaceSetup(t, fooPath, overlayFs, false)
	defer closer()

	startPosition := &token.Position{Filename: foo2GoPath, Line: 3, Column: 10}
	referencePositions := []*token.Position{
		&token.Position{Filename: foo1GoPath, Line: 2, Column: 7},
		startPosition,
	}
	testReferences(t, w, startPosition, referencePositions)
}

func Test_Workspace_References_Imported_Func(t *testing.T) {
	src1 := `package foo
	func GetFoo() int {
		return 0
	}`

	src2 := `package bar
	import "../foo"
	func Do() int {
		return foo.GetFoo()
	}`

	rootPath, overlayFs := createOverlay(map[string]string{
		filepath.Join("foo", "foo.go"): src1,
		filepath.Join("bar", "bar.go"): src2,
	})
	barPath := filepath.Join(rootPath, "bar")
	fooGoPath := filepath.Join(rootPath, "foo", "foo.go")
	barGoPath := filepath.Join(rootPath, "bar", "bar.go")

	w, closer := workspaceSetup(t, barPath, overlayFs, false)
	defer closer()

	startPosition := &token.Position{Filename: barGoPath, Line: 4, Column: 14}
	referencePositions := []*token.Position{
		&token.Position{Filename: fooGoPath, Line: 2, Column: 7},
		startPosition,
	}
	testReferences(t, w, startPosition, referencePositions)
}

func Test_Workspace_References_Local_Selector_Field(t *testing.T) {
	src1 := `package foo
	type fooStruct struct {
		a int
	}
	func Do() int {
		f := &fooStruct{}
		return f.a
	}`

	rootPath, overlayFs := createOverlay(map[string]string{
		filepath.Join("foo", "foo.go"): src1,
	})
	fooPath := filepath.Join(rootPath, "foo")
	fooGoPath := filepath.Join(rootPath, "foo", "foo.go")

	w, closer := workspaceSetup(t, fooPath, overlayFs, false)
	defer closer()

	startPosition := &token.Position{Filename: fooGoPath, Line: 7, Column: 12}
	referencePositions := []*token.Position{
		&token.Position{Filename: fooGoPath, Line: 3, Column: 3},
		startPosition,
	}
	testReferences(t, w, startPosition, referencePositions)
}

func Test_Workspace_References_Package_Selector_Field(t *testing.T) {
	src1 := `package foo
	type fooStruct struct {
		a int
	}`
	src2 := `package foo
	func Do() int {
		f := &fooStruct{}
		return f.a
	}`

	rootPath, overlayFs := createOverlay(map[string]string{
		filepath.Join("foo", "foo1.go"): src1,
		filepath.Join("foo", "foo2.go"): src2,
	})
	fooPath := filepath.Join(rootPath, "foo")
	foo1GoPath := filepath.Join(rootPath, "foo", "foo1.go")
	foo2GoPath := filepath.Join(rootPath, "foo", "foo2.go")

	w, closer := workspaceSetup(t, fooPath, overlayFs, false)
	defer closer()

	startPosition := &token.Position{Filename: foo2GoPath, Line: 4, Column: 12}
	referencePositions := []*token.Position{
		&token.Position{Filename: foo1GoPath, Line: 3, Column: 3},
		startPosition,
	}
	testReferences(t, w, startPosition, referencePositions)
}

func Test_Workspace_References_Imported_Selector_Field(t *testing.T) {
	src1 := `package foo
	type FooStruct struct {
		A int
	}`
	src2 := `package bar
	import "../foo"
	func Do() int {
		f := &foo.FooStruct{}
		return f.A
	}`

	rootPath, overlayFs := createOverlay(map[string]string{
		filepath.Join("foo", "foo.go"): src1,
		filepath.Join("bar", "bar.go"): src2,
	})
	barPath := filepath.Join(rootPath, "bar")
	fooGoPath := filepath.Join(rootPath, "foo", "foo.go")
	barGoPath := filepath.Join(rootPath, "bar", "bar.go")

	w, closer := workspaceSetup(t, barPath, overlayFs, false)
	defer closer()

	startPosition := &token.Position{Filename: barGoPath, Line: 5, Column: 12}
	referencePositions := []*token.Position{
		&token.Position{Filename: fooGoPath, Line: 3, Column: 3},
		startPosition,
	}
	testReferences(t, w, startPosition, referencePositions)
}

func Test_Workspace_References_Indirect_Imported_Selector_Field(t *testing.T) {
	src1 := `package foo
	type FooStruct struct {
		A int
	}`
	src2 := `package bar
	import "../foo"
	type BarStruct struct {
		F *foo.FooStruct
	}
	func NewBarStruct() *BarStruct {
		return &BarStruct {
			F: &foo.FooStruct{}
		}
	}`
	src3 := `package baz
	import "../bar"
	func Do() int {
		b := bar.NewBarStruct()
		return b.F.A
	}`

	rootPath, overlayFs := createOverlay(map[string]string{
		filepath.Join("foo", "foo.go"): src1,
		filepath.Join("bar", "bar.go"): src2,
		filepath.Join("baz", "baz.go"): src3,
	})
	bazPath := filepath.Join(rootPath, "baz")
	bazGoPath := filepath.Join(rootPath, "baz", "baz.go")
	fooGoPath := filepath.Join(rootPath, "foo", "foo.go")

	w, closer := workspaceSetup(t, bazPath, overlayFs, false)
	defer closer()

	startPosition := &token.Position{Filename: bazGoPath, Line: 5, Column: 14}
	referencePositions := []*token.Position{
		&token.Position{Filename: fooGoPath, Line: 3, Column: 3},
		startPosition,
	}
	testReferences(t, w, startPosition, referencePositions)
}

func Test_Workspace_References_Local_Selector_Method(t *testing.T) {
	src1 := `package foo
	type fooStruct struct {}
	func (f *fooStruct) getFoo() int {
		return 0
	}
	func Do() int {
		f := &fooStruct{}
		return f.getFoo()
	}`

	rootPath, overlayFs := createOverlay(map[string]string{
		filepath.Join("foo", "foo.go"): src1,
	})
	fooPath := filepath.Join(rootPath, "foo")
	fooGoPath := filepath.Join(rootPath, "foo", "foo.go")

	w, closer := workspaceSetup(t, fooPath, overlayFs, false)
	defer closer()

	startPosition := &token.Position{Filename: fooGoPath, Line: 8, Column: 12}
	referencePositions := []*token.Position{
		&token.Position{Filename: fooGoPath, Line: 3, Column: 22},
		startPosition,
	}
	testReferences(t, w, startPosition, referencePositions)
}

func Test_Workspace_References_Package_Selector_Method(t *testing.T) {
	src1 := `package foo
	type fooStruct struct {}
	func (f *fooStruct) getFoo() int {
		return 0
	}`
	src2 := `package foo
	func Do() int {
		f := &fooStruct{}
		return f.getFoo()
	}`

	rootPath, overlayFs := createOverlay(map[string]string{
		filepath.Join("foo", "foo1.go"): src1,
		filepath.Join("foo", "foo2.go"): src2,
	})
	fooPath := filepath.Join(rootPath, "foo")
	foo1GoPath := filepath.Join(rootPath, "foo", "foo1.go")
	foo2GoPath := filepath.Join(rootPath, "foo", "foo2.go")

	w, closer := workspaceSetup(t, fooPath, overlayFs, false)
	defer closer()

	startPosition := &token.Position{Filename: foo2GoPath, Line: 4, Column: 12}
	referencePositions := []*token.Position{
		&token.Position{Filename: foo1GoPath, Line: 3, Column: 22},
		startPosition,
	}
	testReferences(t, w, startPosition, referencePositions)
}

func Test_Workspace_References_Imported_Selector_Method(t *testing.T) {
	src1 := `package foo
	type FooStruct struct {}
	func (f *FooStruct) GetFoo() int {
		return 0
	}`
	src2 := `package bar
	import "../foo"
	func Do() int {
		f := &foo.FooStruct{}
		return f.GetFoo()
	}`

	rootPath, overlayFs := createOverlay(map[string]string{
		filepath.Join("foo", "foo.go"): src1,
		filepath.Join("bar", "bar.go"): src2,
	})
	barPath := filepath.Join(rootPath, "bar")
	fooGoPath := filepath.Join(rootPath, "foo", "foo.go")
	barGoPath := filepath.Join(rootPath, "bar", "bar.go")

	w, closer := workspaceSetup(t, barPath, overlayFs, false)
	defer closer()

	startPosition := &token.Position{Filename: barGoPath, Line: 5, Column: 12}
	referencePositions := []*token.Position{
		&token.Position{Filename: fooGoPath, Line: 3, Column: 22},
		startPosition,
	}
	testReferences(t, w, startPosition, referencePositions)
}

func Test_Workspace_References_Indirect_Imported_Selector_Method(t *testing.T) {
	src1 := `package foo
	type FooStruct struct {}
	func (f *FooStruct) GetFoo() int {
		return 0
	}`
	src2 := `package bar
	import "../foo"
	type BarStruct struct {
		F *foo.FooStruct
	}
	func NewBarStruct() *BarStruct {
		return &BarStruct {
			F: &foo.FooStruct{}
		}
	}`
	src3 := `package baz
	import "../bar"
	func Do() int {
		b := bar.NewBarStruct()
		return b.F.GetFoo()
	}`

	rootPath, overlayFs := createOverlay(map[string]string{
		filepath.Join("foo", "foo.go"): src1,
		filepath.Join("bar", "bar.go"): src2,
		filepath.Join("baz", "baz.go"): src3,
	})
	bazPath := filepath.Join(rootPath, "baz")
	bazGoPath := filepath.Join(rootPath, "baz", "baz.go")
	fooGoPath := filepath.Join(rootPath, "foo", "foo.go")

	w, closer := workspaceSetup(t, bazPath, overlayFs, false)
	defer closer()

	startPosition := &token.Position{Filename: bazGoPath, Line: 5, Column: 14}
	referencePositions := []*token.Position{
		&token.Position{Filename: fooGoPath, Line: 3, Column: 22},
		startPosition,
	}
	testReferences(t, w, startPosition, referencePositions)
}

func Test_Workspace_References_Local_Selector_Interface_Method(t *testing.T) {
	src1 := `package foo
	type fooIface interface {
		getFoo() int
	}
	type fooStruct struct {}
	func (f *fooStruct) getFoo() int {
		return 0
	}
	func Do() int {
		f := &fooStruct{}
		i := fooIface(f)
		return i.getFoo()
	}`

	rootPath, overlayFs := createOverlay(map[string]string{
		filepath.Join("foo", "foo.go"): src1,
	})
	fooPath := filepath.Join(rootPath, "foo")
	fooGoPath := filepath.Join(rootPath, "foo", "foo.go")

	w, closer := workspaceSetup(t, fooPath, overlayFs, false)
	defer closer()

	startPosition := &token.Position{Filename: fooGoPath, Line: 12, Column: 12}
	referencePositions := []*token.Position{
		&token.Position{Filename: fooGoPath, Line: 3, Column: 3},
		startPosition,
	}
	testReferences(t, w, startPosition, referencePositions)
}

func Test_Workspace_References_Package_Selector_Interface_Method(t *testing.T) {
	src1 := `package foo
	type fooIface interface {
		getFoo() int
	}
	type fooStruct struct {}
	func (f *fooStruct) getFoo() int {
		return 0
	}`
	src2 := `package foo
	func Do() int {
		f := &fooStruct{}
		i := fooIface(f)
		return i.getFoo()
	}`

	rootPath, overlayFs := createOverlay(map[string]string{
		filepath.Join("foo", "foo1.go"): src1,
		filepath.Join("foo", "foo2.go"): src2,
	})
	fooPath := filepath.Join(rootPath, "foo")
	foo1GoPath := filepath.Join(rootPath, "foo", "foo1.go")
	foo2GoPath := filepath.Join(rootPath, "foo", "foo2.go")

	w, closer := workspaceSetup(t, fooPath, overlayFs, false)
	defer closer()

	startPosition := &token.Position{Filename: foo2GoPath, Line: 5, Column: 12}
	referencePositions := []*token.Position{
		&token.Position{Filename: foo1GoPath, Line: 3, Column: 3},
		startPosition,
	}
	testReferences(t, w, startPosition, referencePositions)
}

func Test_Workspace_References_Imported_Selector_Interface_Method(t *testing.T) {
	src1 := `package foo
	type FooIface interface {
		GetFoo() int
	}
	type FooStruct struct {}
	func (f *FooStruct) GetFoo() int {
		return 0
	}`
	src2 := `package bar
	import "../foo"
	func Do() int {
		f := &foo.FooStruct{}
		i := foo.FooIface(f)
		return i.GetFoo()
	}`

	rootPath, overlayFs := createOverlay(map[string]string{
		filepath.Join("foo", "foo.go"): src1,
		filepath.Join("bar", "bar.go"): src2,
	})
	barPath := filepath.Join(rootPath, "bar")
	fooGoPath := filepath.Join(rootPath, "foo", "foo.go")
	barGoPath := filepath.Join(rootPath, "bar", "bar.go")

	w, closer := workspaceSetup(t, barPath, overlayFs, false)
	defer closer()

	startPosition := &token.Position{Filename: barGoPath, Line: 6, Column: 12}
	referencePositions := []*token.Position{
		&token.Position{Filename: fooGoPath, Line: 3, Column: 3},
		startPosition,
	}
	testReferences(t, w, startPosition, referencePositions)
}

func Test_Workspace_References_Complex_Lookup(t *testing.T) {
	src1 := `package foo
	type FooStruct struct {
		a int
	}
	func (f *FooStruct) GetFoo() int {
		f.a++
		return f.a
	}`
	src2 := `package bar
	import "../foo"
	type BarStruct struct {
		F *foo.FooStruct
	}
	func NewBarStruct() *BarStruct {
		return &BarStruct {
			F: &foo.FooStruct{}
		}
	}
	func (b *BarStruct) DoBar() int {
		return b.F.GetFoo() + b.F.GetFoo()
	}`
	src3 := `package baz
	import "../bar"
	func Do() int {
		b := bar.NewBarStruct()
		return b.F.GetFoo()
	}
	func ExtraDo(b *bar.BarStruct) int {
		return b.DoBar() * b.F.GetFoo()
	}`

	rootPath, overlayFs := createOverlay(map[string]string{
		filepath.Join("foo", "foo.go"): src1,
		filepath.Join("bar", "bar.go"): src2,
		filepath.Join("baz", "baz.go"): src3,
	})
	barGoPath := filepath.Join(rootPath, "bar", "bar.go")
	bazPath := filepath.Join(rootPath, "baz")
	bazGoPath := filepath.Join(rootPath, "baz", "baz.go")
	fooGoPath := filepath.Join(rootPath, "foo", "foo.go")

	w, closer := workspaceSetup(t, bazPath, overlayFs, false)
	defer closer()

	startPosition := &token.Position{Filename: bazGoPath, Line: 5, Column: 14}
	referencePositions := []*token.Position{
		&token.Position{Filename: fooGoPath, Line: 5, Column: 22},
		startPosition,
		&token.Position{Filename: bazGoPath, Line: 8, Column: 26},
		&token.Position{Filename: barGoPath, Line: 12, Column: 14},
		&token.Position{Filename: barGoPath, Line: 12, Column: 29},
	}
	testReferences(t, w, startPosition, referencePositions)
}

func testReferences(t *testing.T, w *Workspace, startPosition *token.Position, referencePositions []*token.Position) {
	// Ensure that the file at for startPosition is open.  We will use our
	// override of the build.Context to get the file contents
	rc, err := w.Loader.context.OpenFile(startPosition.Filename)
	if err != nil {
		t.Fatalf("Failed to open file %s\n\t%s\n", startPosition.Filename, err)
	}
	b, err := ioutil.ReadAll(rc)
	if err != nil {
		t.Fatalf("Failed while attempting to read pseudo-file %s\n\t%s", startPosition.Filename, err.Error())
	}
	w.OpenFile(startPosition.Filename, string(b))

	w.Loader.Wait()

	actual := w.LocateReferences(startPosition)
	if nil == actual {
		t.Fatal("Got nil back")
	}

	report := func() {
		exs := make([]string, len(referencePositions))
		for k, v := range referencePositions {
			exs[k] = v.String()
		}
		exss := strings.Join(exs, "\n\t")
		acs := make([]string, len(actual))
		for k, v := range actual {
			acs[k] = v.String()
		}
		acss := strings.Join(acs, "\n\t")
		t.Errorf("actuals:\n\t%s\nexpected:\n\t%s", acss, exss)
	}

	if len(actual) != len(referencePositions) {
		report()
		t.Fatalf("Incorrect number of references: got %d, expected %d", len(actual), len(referencePositions))
	}
	for _, v := range referencePositions {
		found := false

		for _, v1 := range actual {
			if comparePosition(&v1, v) {
				found = true
				break
			}
		}

		if !found {
			report()
			t.Fatalf("Did not find %s among expected positions", v.String())
		}
	}
}
