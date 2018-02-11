package langd

import (
	"go/token"
	"strings"
	"testing"
)

func Test_Workspace_References_Local_Definition(t *testing.T) {
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

func Test_Workspace_References_Package_Definition(t *testing.T) {
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
