package langd

import (
	"go/token"
	"testing"
)

const (
	test = 2
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

func Test_Workspace_Hover_Local_Basic_Var(t *testing.T) {
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

	done := w.Loader.Start()

	if err := w.OpenFile("/go/src/foo/foo.go", src1); err != nil {
		t.Fatalf("Error while opening file: %s", err.Error())
	}

	<-done

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
