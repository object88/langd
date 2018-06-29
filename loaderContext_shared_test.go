package langd

import (
	"runtime"
	"sync"
	"testing"

	"github.com/object88/langd/collections"
	"golang.org/x/tools/go/buildutil"
)

// Test_LoaderContext_Shared_Package is checked to make sure that a distinct
// package that shares OS/Arch is only loaded once and is commonly referenced
func Test_LoaderContext_Shared_Package(t *testing.T) {
	srcFoo := `package foo
	var FooVal int`

	srcBar := `package bar
	import "../foo"

	func DoBar() int {
		foo.FooVal += 1
		return foo.FooVal
	}`

	srcBaz := `package baz
	import "../foo"

	func DoBaz() int {
		foo.FooVal += 1
		return foo.FooVal
	}`

	packages := map[string]map[string]string{
		"foo": map[string]string{
			"foo.go": srcFoo,
		},
		"bar": map[string]string{
			"bar.go": srcBar,
		},
		"baz": map[string]string{
			"baz.go": srcBaz,
		},
	}

	loader := NewLoader()
	defer loader.Close()

	wg := sync.WaitGroup{}
	wg.Add(2)

	paths := []string{
		"/go/src/bar",
		"/go/src/baz",
	}

	lcs := make([]*LoaderContext, 2)

	for i := 0; i < 2; i++ {
		ii := i
		go func() {
			lc := NewLoaderContext(loader, paths[ii], runtime.GOOS, runtime.GOARCH, "/go", func(lc *LoaderContext) {
				lc.context = buildutil.FakeContext(packages)
			})
			lcs[ii] = lc

			err := lc.LoadDirectory(paths[ii])
			if err != nil {
				t.Fatalf("Error while loading: %s", err.Error())
			}

			lc.Wait()
			wg.Done()
		}()
	}

	wg.Wait()

	errCount := 0
	for i := 0; i < 2; i++ {
		lcs[i].Errors(func(file string, errs []FileError) {
			if errCount == 0 {
				t.Errorf("Loading error in %s:\n", file)
			}
			for k, err := range errs {
				t.Errorf("\t%d: %s\n", k, err.Message)
			}
			errCount++
		})
	}

	if errCount != 0 {
		t.Fatalf("Found %d errors", errCount)
	}

	count := 0
	pkgs := map[string]int{
		"/go/src/foo": 0,
		"/go/src/bar": 0,
		"/go/src/baz": 0,
	}
	loader.caravan.Iter(func(_ collections.Hash, node *collections.Node) bool {
		dp := node.Element.(*DistinctPackage)
		if _, ok := pkgs[dp.Package.AbsPath]; !ok {
			t.Errorf("Found errant package '%s'", dp.Package.AbsPath)
			return true
		}
		pkgs[dp.Package.AbsPath]++
		count++
		return true
	})

	if count != 3 {
		t.Fatalf("Expected 3 distinct packages; have %d", count)
	}

	for path, count := range pkgs {
		if count != 1 {
			t.Errorf("Known package '%s' has reference count %d; expected 1", path, count)
		}
	}
}
