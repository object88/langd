package langd

import (
	"go/token"
	"path/filepath"
	"testing"
)

func Test_Workspace_Declaration_Import_Const(t *testing.T) {
	src1 := `package foo
	const (
		Fooval = 1
	)`

	src2 := `package bar
	import (
		"../foo"
	)
	func fooer() int { 
		return foo.Fooval
	}`

	rootPath, overlayFs := createOverlay(map[string]string{
		filepath.Join("foo", "foo.go"): src1,
		filepath.Join("bar", "bar.go"): src2,
	})
	barPath := filepath.Join(rootPath, "bar")
	barGoPath := filepath.Join(rootPath, "bar", "bar.go")
	fooGoPath := filepath.Join(rootPath, "foo", "foo.go")

	w, closer := workspaceSetup(t, barPath, overlayFs, false)
	defer closer()

	usagePosition := &token.Position{Filename: barGoPath, Line: 6, Column: 14}
	declPosition := &token.Position{Filename: fooGoPath, Line: 3, Column: 3}
	testDeclaration(t, w, usagePosition, declPosition)
}

func Test_Workspace_Declaration_Import_Func(t *testing.T) {
	src1 := `package foo
	func FooFunc() int {
		return 1
	}`

	src2 := `package bar
	import (
		"../foo"
	)
	func fooer() int { 
		return foo.FooFunc()
	}`

	rootPath, overlayFs := createOverlay(map[string]string{
		filepath.Join("foo", "foo.go"): src1,
		filepath.Join("bar", "bar.go"): src2,
	})
	barPath := filepath.Join(rootPath, "bar")
	barGoPath := filepath.Join(rootPath, "bar", "bar.go")
	fooGoPath := filepath.Join(rootPath, "foo", "foo.go")

	w, closer := workspaceSetup(t, barPath, overlayFs, false)
	defer closer()

	usagePosition := &token.Position{Filename: barGoPath, Line: 6, Column: 14}
	declPosition := &token.Position{Filename: fooGoPath, Line: 2, Column: 7}
	testDeclaration(t, w, usagePosition, declPosition)
}
