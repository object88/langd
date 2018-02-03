package langd

import (
	"fmt"
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
	}
	`

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

	declOffset := nthIndex(src1, "ival", 0)
	usageOffset := nthIndex(src1, "ival", 1)
	declPosition := w.Loader.Fset.Position(token.Pos(declOffset + 1))
	usagePosition := w.Loader.Fset.Position(token.Pos(usageOffset + 1))
	pos, err := w.LocateDeclaration(&usagePosition)
	if err != nil {
		t.Errorf(err.Error())
	}
	if declPosition.Offset != pos.Offset {
		t.Fatalf("Incorrect declaration position: expected %d, got %d\n", declPosition.Offset, pos.Offset)
	}

	if err := w.ChangeFile("/go/src/foo/foo.go", 2, 6, 2, 10, "foos"); err != nil {
		t.Errorf(err.Error())
	}

	rope := w.Loader.openedFiles["/go/src/foo/foo.go"]
	ropeString := rope.String()

	if strings.Contains(ropeString, "foof") {
		t.Errorf("Changed file still contains foof:\n%s\n", ropeString)
	}
	if !strings.Contains(ropeString, "foos") {
		t.Errorf("Changed file does not contain foos:\n%s\n", ropeString)
	}

	<-done

	pos, err = w.LocateDeclaration(&usagePosition)
	if err != nil {
		t.Errorf(err.Error())
	}
	if declPosition.Offset != pos.Offset {
		t.Fatalf("Incorrect declaration position: expected %d, got %d\n", declPosition.Offset, pos.Offset)
	}

	// t.Error("Dump")
}

func Test_Workspace_Modify_Cross_File(t *testing.T) {
	src1 := `package foo
	func foof() int {
		ival += 1
		return ival
	}
	`

	src2 := `package foo
	var intval int = 100
	`

	packages := map[string]map[string]string{
		"foo": map[string]string{
			"foo1.go": src1,
			"foo2.go": src2,
		},
	}

	w := workspaceSetup(t, "/go/src/foo", packages, true)

	done := w.Loader.Start()

	if err := w.OpenFile("/go/src/foo/foo2.go", src2); err != nil {
		t.Fatalf("Error while opening file: %s", err.Error())
	}

	<-done

	// Change the definition to reflect what was used in
	if err := w.ChangeFile("/go/src/foo/foo2.go", 1, 5, 1, 11, "ival"); err != nil {
		t.Errorf(err.Error())
	}

	<-done

	fmt.Printf("foo2.go:\n%s\n", w.Loader.openedFiles["/go/src/foo/foo2.go"].String())

	// n, _ := w.Loader.caravan.Find("/go/src/foo")
	// pkg := n.Element.(*Package)
	// pkg.

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
	declPosition, err := w.LocateDeclaration(usagePosition)
	if err != nil {
		t.Fatalf("Error while finding declaration: %s", err.Error())
	}
	if declPosition == nil {
		t.Fatalf("declPosition is nil")
	}
	if !declPosition.IsValid() {
		t.Fatalf("declPosition is not valid")
	}
	if declPosition.Filename != "/go/src/foo/foo2.go" {
		t.Fatalf("Incorrect filename: '%s'", declPosition.Filename)
	}
}
