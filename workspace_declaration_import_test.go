package langd

import (
	"go/token"
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

	packages := map[string]map[string]string{
		"foo": map[string]string{
			"foo.go": src1,
		},
		"bar": map[string]string{
			"bar.go": src2,
		},
	}

	w, _ := workspaceSetup(t, "/go/src/bar", packages, false)

	usagePosition := &token.Position{
		Filename: "/go/src/bar/bar.go",
		Line:     6,
		Column:   14,
	}
	declPosition := &token.Position{
		Filename: "/go/src/foo/foo.go",
		Line:     3,
		Column:   3,
	}
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

	packages := map[string]map[string]string{
		"foo": map[string]string{
			"foo.go": src1,
		},
		"bar": map[string]string{
			"bar.go": src2,
		},
	}

	w, _ := workspaceSetup(t, "/go/src/bar", packages, false)

	usagePosition := &token.Position{
		Filename: "/go/src/bar/bar.go",
		Line:     6,
		Column:   14,
	}
	declPosition := &token.Position{
		Filename: "/go/src/foo/foo.go",
		Line:     2,
		Column:   7,
	}
	testDeclaration(t, w, usagePosition, declPosition)
}
