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

	declOffset := nthIndex(src1, "fooval", 0)
	usageOffset := nthIndex(src1, "fooval", 1)
	test(t, w, declOffset, usageOffset)
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

	declOffset := nthIndex(src1, "fooer", 0)
	usageOffset := nthIndex(src1, "fooer", 1)
	test(t, w, declOffset, usageOffset)
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

	declOffset := nthIndex(src1, "fooval", 0)
	usageOffset := nthIndex(src1, "fooval", 1)
	test(t, w, declOffset, usageOffset)
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

	declOffset := nthIndex(src1, "fooval", 1)
	usageOffset := nthIndex(src1, "fooval", 2)
	test(t, w, declOffset, usageOffset)
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

	w.AssignAST()

	if expectFailure {
		panic("Write this.")
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

func test(t *testing.T, w *Workspace, declOffset, usageOffset int) {
	usagePosition := w.Loader.Fset.Position(token.Pos(usageOffset + 1))
	declPosition, err := w.LocateDeclaration(&usagePosition)
	if err != nil {
		t.Fatal(err.Error())
	}

	if !declPosition.IsValid() {
		t.Fatalf("Returned position '%s' is not valid", declPosition.String())
	}

	if declPosition.Offset != declOffset {
		t.Fatalf("Incorrect decl position: expected %d, got %d\n", declOffset, declPosition.Offset)
	}
}
