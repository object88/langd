package langd

import (
	"go/token"
	"strings"
	"testing"
)

func Test_Workspace_References_Local_Var(t *testing.T) {
	src1 := `package foo
	var fooVal int = 0
	func IncFoo() int {
		fooVal++
		return fooVal
	}`

	packages := map[string]map[string]string{
		"foo": map[string]string{
			"foo.go": src1,
		},
	}

	w := workspaceSetup(t, "/go/src/foo", packages, false)

	startPosition := &token.Position{
		Filename: "/go/src/foo/foo.go",
		Line:     4,
		Column:   3,
	}
	referencePositions := []*token.Position{
		&token.Position{
			Filename: "/go/src/foo/foo.go",
			Line:     2,
			Column:   6,
		},
		startPosition,
		&token.Position{
			Filename: "/go/src/foo/foo.go",
			Line:     5,
			Column:   10,
		},
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

	packages := map[string]map[string]string{
		"foo": map[string]string{
			"foo1.go": src1,
			"foo2.go": src2,
		},
	}

	w := workspaceSetup(t, "/go/src/foo", packages, false)

	startPosition := &token.Position{
		Filename: "/go/src/foo/foo2.go",
		Line:     3,
		Column:   3,
	}
	referencePositions := []*token.Position{
		&token.Position{
			Filename: "/go/src/foo/foo1.go",
			Line:     2,
			Column:   6,
		},
		startPosition,
		&token.Position{
			Filename: "/go/src/foo/foo2.go",
			Line:     4,
			Column:   10,
		},
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

	packages := map[string]map[string]string{
		"foo": map[string]string{
			"foo.go": src1,
		},
		"bar": map[string]string{
			"bar.go": src2,
		},
	}

	w := workspaceSetup(t, "/go/src/bar", packages, false)

	startPosition := &token.Position{
		Filename: "/go/src/bar/bar.go",
		Line:     4,
		Column:   14,
	}
	referencePositions := []*token.Position{
		&token.Position{
			Filename: "/go/src/foo/foo.go",
			Line:     2,
			Column:   6,
		},
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

	packages := map[string]map[string]string{
		"foo": map[string]string{
			"foo.go": src1,
		},
	}

	w := workspaceSetup(t, "/go/src/foo", packages, false)

	startPosition := &token.Position{
		Filename: "/go/src/foo/foo.go",
		Line:     6,
		Column:   8,
	}
	referencePositions := []*token.Position{
		&token.Position{
			Filename: "/go/src/foo/foo.go",
			Line:     2,
			Column:   7,
		},
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

	packages := map[string]map[string]string{
		"foo": map[string]string{
			"foo1.go": src1,
			"foo2.go": src2,
		},
	}

	w := workspaceSetup(t, "/go/src/foo", packages, false)

	startPosition := &token.Position{
		Filename: "/go/src/foo/foo2.go",
		Line:     3,
		Column:   8,
	}
	referencePositions := []*token.Position{
		&token.Position{
			Filename: "/go/src/foo/foo1.go",
			Line:     2,
			Column:   7,
		},
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

	packages := map[string]map[string]string{
		"foo": map[string]string{
			"foo.go": src1,
		},
		"bar": map[string]string{
			"bar.go": src2,
		},
	}

	w := workspaceSetup(t, "/go/src/bar", packages, false)

	startPosition := &token.Position{
		Filename: "/go/src/bar/bar.go",
		Line:     4,
		Column:   12,
	}
	referencePositions := []*token.Position{
		&token.Position{
			Filename: "/go/src/foo/foo.go",
			Line:     2,
			Column:   7,
		},
		startPosition,
	}
	testReferences(t, w, startPosition, referencePositions)
}

func Test_Workspace_References_Local_Func(t *testing.T) {
	src := `package foo
	func getFoo() int {
		return 0
	}
	func Do() int {
		return getFoo()
	}`

	packages := map[string]map[string]string{
		"foo": map[string]string{
			"foo.go": src,
		},
	}

	w := workspaceSetup(t, "/go/src/foo", packages, false)

	startPosition := &token.Position{
		Filename: "/go/src/foo/foo.go",
		Line:     6,
		Column:   10,
	}
	referencePositions := []*token.Position{
		&token.Position{
			Filename: "/go/src/foo/foo.go",
			Line:     2,
			Column:   7,
		},
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

	packages := map[string]map[string]string{
		"foo": map[string]string{
			"foo1.go": src1,
			"foo2.go": src2,
		},
	}

	w := workspaceSetup(t, "/go/src/foo", packages, false)

	startPosition := &token.Position{
		Filename: "/go/src/foo/foo2.go",
		Line:     3,
		Column:   10,
	}
	referencePositions := []*token.Position{
		&token.Position{
			Filename: "/go/src/foo/foo1.go",
			Line:     2,
			Column:   7,
		},
		startPosition,
	}
	testReferences(t, w, startPosition, referencePositions)
}

func Test_Workspace_References_Imported_Package_Func(t *testing.T) {
	src1 := `package foo
	func GetFoo() int {
		return 0
	}`

	src2 := `package bar
	import "../foo"
	func Do() int {
		return foo.GetFoo()
	}`

	packages := map[string]map[string]string{
		"foo": map[string]string{
			"foo.go": src1,
		},
		"bar": map[string]string{
			"bar.go": src2,
		},
	}

	w := workspaceSetup(t, "/go/src/bar", packages, false)

	startPosition := &token.Position{
		Filename: "/go/src/bar/bar.go",
		Line:     4,
		Column:   14,
	}
	referencePositions := []*token.Position{
		&token.Position{
			Filename: "/go/src/foo/foo.go",
			Line:     2,
			Column:   7,
		},
		startPosition,
	}
	testReferences(t, w, startPosition, referencePositions)
}

func testReferences(t *testing.T, w *Workspace, startPosition *token.Position, referencePositions []*token.Position) {
	actual := w.LocateReferences(startPosition)
	if nil == actual {
		t.Fatal("Got nil back")
	}
	if len(actual) != len(referencePositions) {
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
			t.Fatalf("Did not find %s among expected positions\nactuals:\n\t%s\nexpected:\n\t%s", v.String(), acss, exss)
		}
	}
}
