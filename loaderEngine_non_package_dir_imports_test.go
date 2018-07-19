package langd

import (
	"path/filepath"
	"runtime"
	"testing"

	"github.com/spf13/afero"
)

func Test_Load_PackageWithDifferentDir(t *testing.T) {
	src1 := `package foo
	import "../gobar"
	func add1(param1 int) int {
		bar.TotalCalls++
		return param1+1
	}`

	src2 := `package bar
	var TotalCalls = 0`

	rootPath, overlayFs := createOverlay(map[string]string{
		filepath.Join("foo", "foo.go"):   src1,
		filepath.Join("gobar", "bar.go"): src2,
	})
	fooPath := filepath.Join(rootPath, "foo")

	le := NewLoaderEngine()
	defer le.Close()
	l := NewLoader(le, runtime.GOOS, runtime.GOARCH, runtime.GOROOT(), func(l *Loader) {
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
