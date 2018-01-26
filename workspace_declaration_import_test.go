package langd

import (
	"fmt"
	"testing"

	"github.com/object88/langd/collections"
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

	w.Loader.caravan.Iter(func(s string, n *collections.Node) bool {
		p := n.Element.(*Package)
		for k, v := range p.files {
			fmt.Printf("%s :: %#v\n", k, v)
		}
		return true
	})

	declOffset := nthIndex2(w, "/go/src/foo/foo.go", src1, "Fooval", 0)
	usageOffset := nthIndex2(w, "/go/src/bar/bar.go", src2, "Fooval", 0)
	test(t, w, declOffset, usageOffset)
}
