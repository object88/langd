package langd

import (
	"testing"

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
	loader := NewLoader(func(l *Loader) {
		l.context = fc
	})
	done := loader.Start()
	loader.LoadDirectory("/go/src/foo")
	<-done

	errCount := 0
	loader.Errors(func(file string, errs []FileError) {
		errCount++
	})

	if errCount == 0 {
		t.Fatalf("Loader did not emit any errors")
	}
}
