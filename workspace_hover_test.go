package langd

import (
	"go/token"
	"testing"
)

func Test_Workspace_Hover_Local_Const(t *testing.T) {
	src1 := `package foo
	const fooVal = 0
	func IncFoo() int {
		return fooVal
	}`

	packages := map[string]map[string]string{
		"foo": map[string]string{
			"foo.go": src1,
		},
	}

	w := workspaceSetup(t, "/go/src/foo", packages, false)

	p := &token.Position{
		Filename: "/go/src/foo/foo.go",
		Line:     4,
		Column:   10,
	}
	text, err := w.Hover(p)
	if err != nil {
		t.Fatal(err)
	}
	expected := "const foo.fooVal int = 0"
	if text != expected {
		t.Errorf("Expected '%s', got '%s'", expected, text)
	}
}

func Test_Workspace_Hover_Local_Var_Basic(t *testing.T) {
	src1 := `package foo
	var ival int = 10
	func foof() int {
		ival += 1
		return ival
	}`

	packages := map[string]map[string]string{
		"foo": map[string]string{
			"foo.go": src1,
		},
	}

	w := workspaceSetup(t, "/go/src/foo", packages, false)

	p := &token.Position{
		Filename: "/go/src/foo/foo.go",
		Line:     4,
		Column:   3,
	}
	text, err := w.Hover(p)
	if err != nil {
		t.Fatal(err)
	}
	expected := "foo.ival int"
	if text != expected {
		t.Errorf("Expected '%s', got '%s'", expected, text)
	}
}

func Test_Workspace_Hover_Local_Var_Struct_Empty(t *testing.T) {
	src1 := `package foo
	type fooer struct {
	}
	var ival fooer
	func foof() fooer {
		return ival
	}`

	packages := map[string]map[string]string{
		"foo": map[string]string{
			"foo.go": src1,
		},
	}

	w := workspaceSetup(t, "/go/src/foo", packages, false)

	p := &token.Position{
		Filename: "/go/src/foo/foo.go",
		Line:     6,
		Column:   10,
	}
	text, err := w.Hover(p)
	if err != nil {
		t.Fatal(err)
	}
	expected := "type foo.fooer struct {}"
	if text != expected {
		t.Errorf("Expected '%s', got '%s'", expected, text)
	}
}

func Test_Workspace_Hover_Local_Var_Struct_With_Fields(t *testing.T) {
	src1 := `package foo
	type fooer struct {
		a int
		b string
	}
	var ival fooer
	func foof() fooer {
		ival.a += 1
		return ival
	}`

	packages := map[string]map[string]string{
		"foo": map[string]string{
			"foo.go": src1,
		},
	}

	w := workspaceSetup(t, "/go/src/foo", packages, false)

	p := &token.Position{
		Filename: "/go/src/foo/foo.go",
		Line:     8,
		Column:   3,
	}
	text, err := w.Hover(p)
	if err != nil {
		t.Fatal(err)
	}
	expected := "type foo.fooer struct {\n\ta int\n\tb string\n}"
	if text != expected {
		t.Errorf("Expected '%s', got '%s'", expected, text)
	}
}

func Test_Workspace_Hover_Local_Var_Struct_Embedded(t *testing.T) {
	src1 := `package foo
	type fooer struct {
		a int
		b string
	}
	type barer struct {
		fooer
		c float32
	}
	var ival barer
	func foof() barer {
		ival.c += 1
		return ival
	}`

	packages := map[string]map[string]string{
		"foo": map[string]string{
			"foo.go": src1,
		},
	}

	w := workspaceSetup(t, "/go/src/foo", packages, false)

	p := &token.Position{
		Filename: "/go/src/foo/foo.go",
		Line:     12,
		Column:   3,
	}
	text, err := w.Hover(p)
	if err != nil {
		t.Fatal(err)
	}
	expected := "type foo.barer struct {\n\tfoo.fooer\n\tc float32\n}"
	if text != expected {
		t.Errorf("Expected '%s', got '%s'", expected, text)
	}
}
