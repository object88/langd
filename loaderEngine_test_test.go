package langd

import (
	"path/filepath"
	"runtime"
	"testing"

	"github.com/spf13/afero"
)

func Test_Loader_Load_Tests(t *testing.T) {
	srcBar := `package bar
	var BarVal int = 0`

	srcBarTest := `package bar
	import "../foo"

	func Test_Bar() {
		BarVal = foo.FooVal
	}
	`

	srcFoo := `package foo
	var FooVal int = 1`

	rootPath, overlayFs := createOverlay(map[string]string{
		filepath.Join("bar", "bar.go"):      srcBar,
		filepath.Join("bar", "bar_test.go"): srcBarTest,
		filepath.Join("foo", "foo.go"):      srcFoo,
	})
	barPath := filepath.Join(rootPath, "bar")

	le := NewLoaderEngine()
	defer le.Close()

	l := NewLoader(le, barPath, "darwin", "x86", runtime.GOROOT(), func(l *Loader) {
		l.fs = afero.NewCopyOnWriteFs(l.fs, overlayFs)
	})

	err := l.LoadDirectory(barPath)
	if err != nil {
		t.Fatalf("(1) Error while loading: %s", err.Error())
	}

	l.Wait()

	errCount := 0
	fn := func(file string, errs []FileError) {
		if errCount == 0 {
			t.Errorf("Loading error in %s:\n", file)
		}
		for k, err := range errs {
			t.Errorf("\t%d: %s\n", k, err.Message)
		}
		errCount++
	}

	l.Errors(fn)

	if errCount != 0 {
		t.Fatalf("Found %d errors", errCount)
	}

}
