package langd

import (
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

	w := workspaceSetup(t, "/go/src/bar", packages, false)

	declOffset := nthIndex2(w, "/go/src/foo/foo.go", src1, "Fooval", 0)
	usageOffset := nthIndex2(w, "/go/src/bar/bar.go", src2, "Fooval", 0)
	test(t, w, declOffset, usageOffset)
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

	w := workspaceSetup(t, "/go/src/bar", packages, false)

	declOffset := nthIndex2(w, "/go/src/foo/foo.go", src1, "FooFunc", 0)
	usageOffset := nthIndex2(w, "/go/src/bar/bar.go", src2, "FooFunc", 0)
	test(t, w, declOffset, usageOffset)
}
