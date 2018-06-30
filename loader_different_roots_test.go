package langd

import "testing"

// Test_LoaderContext_Different_Root will require a different test context
// than FakeContext, which only allows one GOROOT.  We will want to start to
// use Afero (https://github.com/spf13/afero) in the LoaderContext, so that
// we can provide memory-mapped complex file systems.
func Test_LoaderContext_Different_Root(t *testing.T) {
	t.Error("Not implemented")
}
