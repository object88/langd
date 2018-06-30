package langd

import (
	"runtime"
	"testing"

	"github.com/object88/langd/log"
	"golang.org/x/tools/go/buildutil"
)

// Looking to test circumstance where a package gets imported, but the
// workspace may not contain packages that it is importing

func Test_Load_Missing_Imports(t *testing.T) {
	code := `package foo
	import "missing"
	
	var f missing.Thing`

	packages := map[string]map[string]string{
		"foo": map[string]string{
			"foo.go": code,
		},
	}

	fc := buildutil.FakeContext(packages)
	le := NewLoaderEngine()
	defer le.Close()
	l := NewLoader(le, "/go/src/foo", runtime.GOOS, runtime.GOARCH, "/go", func(l *Loader) {
		l.context = fc
		l.Log.SetLevel(log.Debug)
	})

	l.LoadDirectory("/go/src/foo")
	l.Wait()

	errCount := 0
	l.Errors(func(file string, errs []FileError) {
		t.Logf("Got %d errors in %s\n", len(errs), file)
		for _, v := range errs {
			t.Logf("\t%s\n", v.String())
		}
		errCount++
	})

	if errCount == 0 {
		t.Fatalf("Loader did not emit any errors")
	}
}
