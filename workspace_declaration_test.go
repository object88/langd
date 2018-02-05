package langd

import (
	"bytes"
	"fmt"
	"go/token"
	"os"
	"testing"

	"github.com/object88/langd/log"
	"golang.org/x/tools/go/buildutil"
)

func Test_Workspace_Declaration_Package_Const(t *testing.T) {
	src1 := `package foo
	const (
		fooval = 1
	)
	func fooer() int { 
		return fooval
	}`

	packages := map[string]map[string]string{
		"foo": map[string]string{
			"foo.go": src1,
		},
	}

	w := workspaceSetup(t, "/go/src/foo", packages, false)

	usagePosition := &token.Position{
		Filename: "/go/src/foo/foo.go",
		Line:     6,
		Column:   10,
	}
	declPosition := &token.Position{
		Filename: "/go/src/foo/foo.go",
		Line:     3,
		Column:   3,
	}
	test(t, w, usagePosition, declPosition)
}

func Test_Workspace_Declaration_Package_Func(t *testing.T) {
	src1 := `package foo
	func fooer() int { return 0 }
	func init() {
		fooer()
	}`

	packages := map[string]map[string]string{
		"foo": map[string]string{
			"foo.go": src1,
		},
	}

	w := workspaceSetup(t, "/go/src/foo", packages, false)

	usagePosition := &token.Position{
		Filename: "/go/src/foo/foo.go",
		Line:     4,
		Column:   3,
	}
	declPosition := &token.Position{
		Filename: "/go/src/foo/foo.go",
		Line:     2,
		Column:   7,
	}
	test(t, w, usagePosition, declPosition)
}

func Test_Workspace_Declaration_Package_Var(t *testing.T) {
	src1 := `package foo
	var fooval = 1
	func fooer() int { 
		return fooval
	}`

	packages := map[string]map[string]string{
		"foo": map[string]string{
			"foo.go": src1,
		},
	}

	w := workspaceSetup(t, "/go/src/foo", packages, false)

	usagePosition := &token.Position{
		Filename: "/go/src/foo/foo.go",
		Line:     4,
		Column:   10,
	}
	declPosition := &token.Position{
		Filename: "/go/src/foo/foo.go",
		Line:     2,
		Column:   6,
	}
	test(t, w, usagePosition, declPosition)
}

func Test_Workspace_Declaration_Package_Var_CrossFiles(t *testing.T) {
	src1 := `package foo
	func foof() int {
		ival += 1
		return ival
	}`

	src2 := `package foo
	var ival int = 100`

	packages := map[string]map[string]string{
		"foo": map[string]string{
			"foo1.go": src1,
			"foo2.go": src2,
		},
	}

	w := workspaceSetup(t, "/go/src/foo", packages, false)
	usagePosition := &token.Position{
		Filename: "/go/src/foo/foo1.go",
		Line:     3,
		Column:   3,
	}
	declPosition := &token.Position{
		Filename: "/go/src/foo/foo2.go",
		Line:     2,
		Column:   6,
	}
	test(t, w, usagePosition, declPosition)
}

func Test_Workspace_Declaration_Package_Var_Shadowed(t *testing.T) {
	src1 := `package foo
	var fooval = 1
	func fooer() int { 
		fooval := 0
		return fooval
	}`

	packages := map[string]map[string]string{
		"foo": map[string]string{
			"foo.go": src1,
		},
	}

	w := workspaceSetup(t, "/go/src/foo", packages, false)

	declPosition := &token.Position{
		Filename: "/go/src/foo/foo.go",
		Line:     4,
		Column:   3,
	}
	usagePosition := &token.Position{
		Filename: "/go/src/foo/foo.go",
		Line:     5,
		Column:   10,
	}
	test(t, w, usagePosition, declPosition)
}

func workspaceSetup(t *testing.T, startingPath string, packages map[string]map[string]string, expectFailure bool) *Workspace {
	fc := buildutil.FakeContext(packages)
	loader := NewLoader(func(l *Loader) {
		l.context = fc
	})
	w := CreateWorkspace(loader, log.CreateLog(os.Stdout))
	w.log.SetLevel(log.Verbose)

	done := loader.Start()
	loader.LoadDirectory(startingPath)
	<-done

	if expectFailure {
		errCount := 0
		w.Loader.Errors(func(file string, errs []FileError) {
			errCount += len(errs)
		})
		if errCount == 0 {
			t.Fatal("Expected errors, but got none\n")
		}
	} else {
		errCount := 0
		var buf bytes.Buffer
		w.Loader.Errors(func(file string, errs []FileError) {
			buf.WriteString(fmt.Sprintf("Loading error in %s:\n", file))
			for k, err := range errs {
				buf.WriteString(fmt.Sprintf("\t%02d: %s:%d %s\n", k, err.Filename, err.Line, err.Message))
			}
			errCount += len(errs)
		})

		if errCount != 0 {
			buf.WriteString(fmt.Sprintf("Total: %d errors\n", errCount))
			t.Fatal(buf.String())
		}
	}

	return w
}

// func test(t *testing.T, w *Workspace, declOffset, usageOffset int) {
func test(t *testing.T, w *Workspace, usagePosition, expectedDeclPosition *token.Position) {
	declPosition, err := w.LocateDeclaration(usagePosition)
	if err != nil {
		t.Fatal(err.Error())
	}

	comparePosition(t, declPosition, expectedDeclPosition)
}

func comparePosition(t *testing.T, actual, expected *token.Position) {
	if actual == nil {
		t.Fatalf("actual is nil")
	}

	if !actual.IsValid() {
		t.Fatalf("Returned position '%s' is not valid", actual.String())
	}

	if actual.Filename != expected.Filename {
		t.Fatalf("Incorrect filename: got '%s', expected '%s'", actual.Filename, expected.Filename)
	}

	if actual.Line != expected.Line {
		t.Fatalf("Incorrect line: got %d, expected %d", actual.Line, expected.Line)
	}

	if actual.Column != expected.Column {
		t.Fatalf("Incorrect column: got %d, expected %d", actual.Column, expected.Column)
	}

}
