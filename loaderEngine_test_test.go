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

	le := NewLoaderEngine()
	defer le.Close()

	l := NewLoader(le, "/go/src/bar", "darwin", "x86", "/go", func(l *Loader) {
		l.context = buildutil.FakeContext(packages)
	})

	err := l.LoadDirectory("/go/src/bar")
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
