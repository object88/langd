package langd

import (
	"go/token"
	"strings"
	"testing"
)

func Test_Workspace_Modify_File(t *testing.T) {
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

	w, l, closer := workspaceSetup(t, "/go/src/foo", packages, false)
	defer closer()

	if err := w.OpenFile("/go/src/foo/foo.go", src1); err != nil {
		t.Fatalf("Error while opening file: %s", err.Error())
	}

	l.Wait()

	declPosition := &token.Position{
		Filename: "/go/src/foo/foo.go",
		Line:     2,
		Column:   6,
	}
	usagePosition := &token.Position{
		Filename: "/go/src/foo/foo.go",
		Line:     4,
		Column:   3,
	}
	pos, err := w.LocateDeclaration(usagePosition)
	if err != nil {
		t.Errorf(err.Error())
	}
	testPosition(t, pos, declPosition)

	if err := w.ChangeFile("/go/src/foo/foo.go", 2, 6, 2, 10, "foos"); err != nil {
		t.Errorf(err.Error())
	}

	l.Wait()

	rope, _ := w.LoaderEngine.openedFiles.Get("/go/src/foo/foo.go")
	ropeString := rope.String()

	if strings.Contains(ropeString, "foof") {
		t.Errorf("Changed file still contains foof:\n%s\n", ropeString)
	}
	if !strings.Contains(ropeString, "foos") {
		t.Errorf("Changed file does not contain foos:\n%s\n", ropeString)
	}

	l.Wait()

	pos, err = w.LocateDeclaration(usagePosition)
	if err != nil {
		t.Errorf(err.Error())
	}
	testPosition(t, pos, declPosition)
}

func Test_Workspace_Modify_Cross_File(t *testing.T) {
	src1 := `package foo
	func foof() int {
		ival += 1
		return ival
	}`

	src2 := `package foo
	var intval int = 100
	`

	packages := map[string]map[string]string{
		"foo": map[string]string{
			"foo1.go": src1,
			"foo2.go": src2,
		},
	}

	w, l, closer := workspaceSetup(t, "/go/src/foo", packages, true)
	defer closer()

	if err := w.OpenFile("/go/src/foo/foo2.go", src2); err != nil {
		t.Fatalf("Error while opening file: %s", err.Error())
	}

	l.Wait()

	// Change the definition to reflect what was used in
	if err := w.ChangeFile("/go/src/foo/foo2.go", 1, 5, 1, 11, "ival"); err != nil {
		t.Errorf(err.Error())
	}

	l.Wait()

	errCount := 0
	w.Loader.Errors(func(file string, errs []FileError) {
		for _, err := range errs {
			t.Error(err.Message)
		}
		errCount += len(errs)
	})
	if errCount != 0 {
		t.Errorf("Failed to correct type checker errors; have %d errors", errCount)
	}

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
	decl, err := w.LocateDeclaration(usagePosition)
	if err != nil {
		t.Fatalf("Error while finding declaration: %s", err.Error())
	}
	testPosition(t, decl, declPosition)
}
