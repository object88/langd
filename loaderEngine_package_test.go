package langd

import (
	"path/filepath"
	"runtime"
	"testing"

	"github.com/spf13/afero"
)

func Test_Load_Own_Package(t *testing.T) {
	src := `package bar
	import "../bar"
	var Bar int = 0`

	rootPath, overlayFs := createOverlay(map[string]string{
		filepath.Join("bar", "bar.go"): src,
	})
	barPath := filepath.Join(rootPath, "bar")

	le := NewLoaderEngine()
	defer le.Close()
	l := NewLoader(le, runtime.GOOS, runtime.GOARCH, runtime.GOROOT(), func(l *Loader) {
		l.fs = afero.NewCopyOnWriteFs(l.fs, overlayFs)
	})

	err := l.LoadDirectory(barPath)
	if err != nil {
		t.Fatalf("Error while loading: %s", err.Error())
	}
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
