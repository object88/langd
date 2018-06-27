package langd

import (
	"testing"

	"golang.org/x/tools/go/buildutil"
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

	packages := map[string]map[string]string{
		"bar": map[string]string{
			"bar.go":      srcBar,
			"bar_test.go": srcBarTest,
		},
		"foo": map[string]string{
			"foo.go": srcFoo,
		},
	}

	loader := NewLoader()
	defer loader.Close()

	lc := NewLoaderContext(loader, "/go/src/bar", "darwin", "x86", "/go", func(lc LoaderContext) {
		lc.(*loaderContext).context = buildutil.FakeContext(packages)
	})

	err := loader.LoadDirectory(lc, "/go/src/bar")
	if err != nil {
		t.Fatalf("(1) Error while loading: %s", err.Error())
	}

	lc.Wait()

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

	loader.Errors(lc, fn)

	if errCount != 0 {
		t.Fatalf("Found %d errors", errCount)
	}

}
