package langd

import (
	"go/token"
	"path/filepath"
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

	rootPath, overlayFs := createOverlay(map[string]string{
		filepath.Join("foo", "foo.go"): src1,
	})
	fooPath := filepath.Join(rootPath, "foo")
	fooGoPath := filepath.Join(rootPath, "foo", "foo.go")

	w, closer := workspaceSetup(t, fooPath, overlayFs, false)
	defer closer()

	if err := w.OpenFile(fooGoPath, src1); err != nil {
		t.Fatalf("Error while opening file: %s", err.Error())
	}

	w.Loader.Wait()

	declPosition := &token.Position{Filename: fooGoPath, Line: 2, Column: 6}
	usagePosition := &token.Position{Filename: fooGoPath, Line: 4, Column: 3}
	pos, err := w.LocateDeclaration(usagePosition)
	if err != nil {
		t.Errorf(err.Error())
	}
	testPosition(t, pos, declPosition)

	if err := w.ChangeFile(fooGoPath, 2, 6, 2, 10, "foos"); err != nil {
		t.Errorf(err.Error())
	}

	w.Loader.Wait()

	rope, _ := w.LoaderEngine.openedFiles.Get(fooGoPath)
	ropeString := rope.String()

	if strings.Contains(ropeString, "foof") {
		t.Errorf("Changed file still contains foof:\n%s\n", ropeString)
	}
	if !strings.Contains(ropeString, "foos") {
		t.Errorf("Changed file does not contain foos:\n%s\n", ropeString)
	}

	w.Loader.Wait()

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

	rootPath, overlayFs := createOverlay(map[string]string{
		filepath.Join("foo", "foo1.go"): src1,
		filepath.Join("foo", "foo2.go"): src2,
	})
	fooPath := filepath.Join(rootPath, "foo")
	foo1GoPath := filepath.Join(rootPath, "foo", "foo1.go")
	foo2GoPath := filepath.Join(rootPath, "foo", "foo2.go")

	w, closer := workspaceSetup(t, fooPath, overlayFs, true)
	defer closer()

	if err := w.OpenFile(foo2GoPath, src2); err != nil {
		t.Fatalf("Error while opening file: %s", err.Error())
	}

	w.Loader.Wait()

	// Change the definition to reflect what was used in
	if err := w.ChangeFile(foo2GoPath, 1, 5, 1, 11, "ival"); err != nil {
		t.Errorf(err.Error())
	}

	w.Loader.Wait()

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

	usagePosition := &token.Position{Filename: foo1GoPath, Line: 3, Column: 3}
	declPosition := &token.Position{Filename: foo2GoPath, Line: 2, Column: 6}
	decl, err := w.LocateDeclaration(usagePosition)
	if err != nil {
		t.Fatalf("Error while finding declaration: %s", err.Error())
	}
	testPosition(t, decl, declPosition)
}
