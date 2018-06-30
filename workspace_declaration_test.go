package langd

import (
	"go/token"
	"path/filepath"
	"testing"
)

// func Test_Workspace_References_Local_Package(t *testing.T) {
// 	src := `package foo
// 	var fooInt int = 0`

// 	packages := map[string]map[string]string{
// 		"foo": map[string]string{
// 			"foo.go": src,
// 		},
// 	}

// 	w, _, closer := workspaceSetup(t, "/go/src/foo", packages, false)
// defer closer()

// 	startPosition := &token.Position{
// 		Filename: "/go/src/foo/foo.go",
// 		Line:     1,
// 		Column:   9,
// 	}
// 	referencePositions := []*token.Position{
// 		startPosition,
// 	}
// 	testReferences(t, w, startPosition, referencePositions)
// }

func Test_Workspace_Declaration_Imported_Package(t *testing.T) {
	src1 := `package foo
	const MyNumber = 0`

	src2 := `package bar
	import "../foo"
	func Do() int {
		return foo.MyNumber
	}`

	rootPath, overlayFs := createOverlay(map[string]string{
		filepath.Join("foo", "foo.go"): src1,
		filepath.Join("bar", "bar.go"): src2,
	})
	barPath := filepath.Join(rootPath, "bar")
	barGoPath := filepath.Join(rootPath, "bar", "bar.go")

	w, closer := workspaceSetup(t, barPath, overlayFs, false)
	defer closer()

	usagePosition := &token.Position{Filename: barGoPath, Line: 4, Column: 10}
	declPosition := &token.Position{Filename: barGoPath, Line: 2, Column: 9}
	testDeclaration(t, w, usagePosition, declPosition)
}

func Test_Workspace_Declaration_Package_Const(t *testing.T) {
	src1 := `package foo
	const (
		fooval = 1
	)
	func fooer() int { 
		return fooval
	}`

	rootPath, overlayFs := createOverlay(map[string]string{
		filepath.Join("foo", "foo.go"): src1,
	})
	fooPath := filepath.Join(rootPath, "foo")
	fooGoPath := filepath.Join(rootPath, "foo", "foo.go")

	w, closer := workspaceSetup(t, fooPath, overlayFs, false)
	defer closer()

	usagePosition := &token.Position{Filename: fooGoPath, Line: 6, Column: 10}
	declPosition := &token.Position{Filename: fooGoPath, Line: 3, Column: 3}
	testDeclaration(t, w, usagePosition, declPosition)
}

func Test_Workspace_Declaration_Package_Func(t *testing.T) {
	src1 := `package foo
	func fooer() int { return 0 }
	func init() {
		fooer()
	}`

	rootPath, overlayFs := createOverlay(map[string]string{
		filepath.Join("foo", "foo.go"): src1,
	})
	fooPath := filepath.Join(rootPath, "foo")
	fooGoPath := filepath.Join(rootPath, "foo", "foo.go")

	w, closer := workspaceSetup(t, fooPath, overlayFs, false)
	defer closer()

	usagePosition := &token.Position{Filename: fooGoPath, Line: 4, Column: 3}
	declPosition := &token.Position{Filename: fooGoPath, Line: 2, Column: 7}
	testDeclaration(t, w, usagePosition, declPosition)
}

func Test_Workspace_Declaration_Package_Var(t *testing.T) {
	src1 := `package foo
	var fooval = 1
	func fooer() int { 
		return fooval
	}`

	rootPath, overlayFs := createOverlay(map[string]string{
		filepath.Join("foo", "foo.go"): src1,
	})
	fooPath := filepath.Join(rootPath, "foo")
	fooGoPath := filepath.Join(rootPath, "foo", "foo.go")

	w, closer := workspaceSetup(t, fooPath, overlayFs, false)
	defer closer()

	usagePosition := &token.Position{Filename: fooGoPath, Line: 4, Column: 10}
	declPosition := &token.Position{Filename: fooGoPath, Line: 2, Column: 6}
	testDeclaration(t, w, usagePosition, declPosition)
}

func Test_Workspace_Declaration_Package_Var_CrossFiles(t *testing.T) {
	src1 := `package foo
	func foof() int {
		ival += 1
		return ival
	}`

	src2 := `package foo
	var ival int = 100`

	rootPath, overlayFs := createOverlay(map[string]string{
		filepath.Join("foo", "foo1.go"): src1,
		filepath.Join("foo", "foo2.go"): src2,
	})
	fooPath := filepath.Join(rootPath, "foo")
	foo1GoPath := filepath.Join(rootPath, "foo", "foo1.go")
	foo2GoPath := filepath.Join(rootPath, "foo", "foo2.go")

	w, closer := workspaceSetup(t, fooPath, overlayFs, false)
	defer closer()
	usagePosition := &token.Position{Filename: foo1GoPath, Line: 3, Column: 3}
	declPosition := &token.Position{Filename: foo2GoPath, Line: 2, Column: 6}
	testDeclaration(t, w, usagePosition, declPosition)
}

func Test_Workspace_Declaration_Package_Var_Shadowed(t *testing.T) {
	src1 := `package foo
	var fooval = 1
	func fooer() int { 
		fooval := 0
		return fooval
	}`

	rootPath, overlayFs := createOverlay(map[string]string{
		filepath.Join("foo", "foo.go"): src1,
	})
	fooPath := filepath.Join(rootPath, "foo")
	fooGoPath := filepath.Join(rootPath, "foo", "foo.go")

	w, closer := workspaceSetup(t, fooPath, overlayFs, false)
	defer closer()

	declPosition := &token.Position{Filename: fooGoPath, Line: 4, Column: 3}
	usagePosition := &token.Position{Filename: fooGoPath, Line: 5, Column: 10}
	testDeclaration(t, w, usagePosition, declPosition)
}
