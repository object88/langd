package langd

import (
	"path/filepath"
	"runtime"
	"testing"

	"github.com/spf13/afero"

	"github.com/object88/langd/log"
)

// Looking to test circumstance where a package gets imported, but the
// workspace may not contain packages that it is importing

func Test_Load_Missing_Imports(t *testing.T) {
	src := `package foo
	import "missing"
	
	var f missing.Thing`

	rootPath, overlayFs := createOverlay(map[string]string{
		filepath.Join("foo", "foo.go"): src,
	})
	fooPath := filepath.Join(rootPath, "foo")

	le := NewLoaderEngine()
	defer le.Close()
	l := NewLoader(le, fooPath, runtime.GOOS, runtime.GOARCH, runtime.GOROOT(), func(l *Loader) {
		l.fs = afero.NewCopyOnWriteFs(l.fs, overlayFs)
		l.Log.SetLevel(log.Debug)
	})

	l.LoadDirectory(fooPath)
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
