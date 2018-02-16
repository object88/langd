package langd

import (
	"go/token"
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

// 	w := workspaceSetup(t, "/go/src/foo", packages, false)

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

func Test_Workspace_References_Imported_Package(t *testing.T) {
	src1 := `package foo
	const MyNumber = 0`

	src2 := `package bar
	import "../foo"
	func Do() int {
		return foo.MyNumber
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

	usagePosition := &token.Position{
		Filename: "/go/src/bar/bar.go",
		Line:     4,
		Column:   10,
	}
	declPosition := &token.Position{
		Filename: "/go/src/bar/bar.go",
		Line:     2,
		Column:   9,
	}
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
	testDeclaration(t, w, usagePosition, declPosition)
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
	testDeclaration(t, w, usagePosition, declPosition)
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
	testDeclaration(t, w, usagePosition, declPosition)
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
	testDeclaration(t, w, usagePosition, declPosition)
}
