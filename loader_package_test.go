package langd

import (
	"runtime"
	"testing"

	"github.com/object88/langd/log"
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
	loader := NewLoader()
	loader.Log.SetLevel(log.Debug)
	lc := NewLoaderContext(loader, runtime.GOOS, runtime.GOARCH, "/go", func(lc *LoaderContext) {
		lc.context = fc
	})

	done := loader.Start()
	err := loader.LoadDirectory(lc, "/go/src/bar")
	if err != nil {
		t.Fatalf("Error while loading: %s", err.Error())
	}
	<-done

	errCount := 0
	loader.Errors(func(file string, errs []FileError) {
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
