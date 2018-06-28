package langd

import (
	"runtime"
	"testing"

	"golang.org/x/tools/go/buildutil"
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

	packages := map[string]map[string]string{
		"foo": map[string]string{
			"foo.go": src1,
		},
		"gobar": map[string]string{
			"bar.go": src2,
		},
	}

	fc := buildutil.FakeContext(packages)
	loader := NewLoader()
	defer loader.Close()
	lc := NewLoaderContext(loader, "/go/src/foo", runtime.GOOS, runtime.GOARCH, "/go", func(lc *LoaderContext) {
		lc.context = fc
	})
	loader.LoadDirectory(lc, "/go/src/foo")
	lc.Wait()

	errCount := 0
	loader.Errors(lc, func(file string, errs []FileError) {
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
