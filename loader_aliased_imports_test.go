package langd

import (
	"runtime"
	"testing"

	"golang.org/x/tools/go/buildutil"
)

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
}
`

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
}
`

func Test_Load_AliasedImports(t *testing.T) {
	packages := map[string]map[string]string{
		"foo": map[string]string{
			"foo.go": aliasedImportsTestProgram1,
		},
		"bar": map[string]string{
			"bar.go": aliasedImportsTestProgram2,
		},
	}

	fc := buildutil.FakeContext(packages)
	loader := NewLoader()
	lc := NewLoaderContext(loader, runtime.GOOS, runtime.GOARCH, "/go", func(lc *LoaderContext) {
		lc.context = fc
	})
	done := loader.Start()
	loader.LoadDirectory(lc, "/go/src/foo")
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
