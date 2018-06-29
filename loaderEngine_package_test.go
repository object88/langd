package langd

import (
	"runtime"
	"testing"

	"golang.org/x/tools/go/buildutil"
)

func Test_Load_Own_Package(t *testing.T) {
	src := `package bar
	import "../bar"
	var Bar int = 0`

	packages := map[string]map[string]string{
		"bar": map[string]string{
			"bar.go": src,
		},
	}

	fc := buildutil.FakeContext(packages)
	le := NewLoaderEngine()
	defer le.Close()
	lc := NewLoaderContext(le, "/go/src/bar", runtime.GOOS, runtime.GOARCH, "/go", func(lc *LoaderContext) {
		lc.context = fc
	})

	err := lc.LoadDirectory("/go/src/bar")
	if err != nil {
		t.Fatalf("Error while loading: %s", err.Error())
	}
	lc.Wait()

	errCount := 0
	lc.Errors(func(file string, errs []FileError) {
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
