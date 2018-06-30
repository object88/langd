package langd

import (
	"path/filepath"
	"runtime"
	"testing"

	"github.com/spf13/afero"
)

func Test_Load_AliasedImports(t *testing.T) {
	const aliasedImportsTestProgram1 = `package foo

	import (
		baz "../bar"
	)

	func add1(param1 int) int {
		baz.CountCall("add1")

		if baz.TotalCalls == 100 {
			// That's a lot of calls...
			return param1
		}

		param1++
		return param1
	}`

	const aliasedImportsTestProgram2 = `package bar

	var calls = map[string]int{}
	var TotalCalls = 0

	func CountCall(source string) {
		TotalCalls++

		call, ok := calls[source]
		if !ok {
			call = 1
		} else {
			call++
		}
		calls[source] = call
	}`

	rootPath, overlayFs := createOverlay(map[string]string{
		filepath.Join("foo", "foo.go"): aliasedImportsTestProgram1,
		filepath.Join("bar", "bar.go"): aliasedImportsTestProgram2,
	})
	fooPath := filepath.Join(rootPath, "foo")

	le := NewLoaderEngine()
	defer le.Close()
	l := NewLoader(le, fooPath, runtime.GOOS, runtime.GOARCH, runtime.GOROOT(), func(l *Loader) {
		l.fs = afero.NewCopyOnWriteFs(l.fs, overlayFs)
	})
	l.LoadDirectory(fooPath)
	l.Wait()

	errCount := 0
	l.Errors(func(file string, errs []FileError) {
		if errCount == 0 {
			t.Errorf("Loading error in %s:\n", file)
		}
		for k, err := range errs {
			t.Errorf("\t%d: %s\n", k, err.Message)
		}
		errCount++
	})

	if errCount != 0 {
		t.Fatalf("Found %d errors", errCount)
	}
}
