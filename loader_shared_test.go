package langd

import (
	"path/filepath"
	"runtime"
	"sync"
	"testing"

	"github.com/object88/langd/collections"
	"github.com/spf13/afero"
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

	rootPath, overlayFs := createOverlay(map[string]string{
		filepath.Join("foo", "foo.go"): srcFoo,
		filepath.Join("bar", "bar.go"): srcBar,
		filepath.Join("baz", "baz.go"): srcBaz,
	})

	le := NewLoaderEngine()
	defer le.Close()

	wg := sync.WaitGroup{}
	wg.Add(2)

	paths := []string{
		filepath.Join(rootPath, "bar"),
		filepath.Join(rootPath, "baz"),
	}

	ls := make([]*Loader, 2)

	for i := 0; i < 2; i++ {
		ii := i
		go func() {
			l := NewLoader(le, paths[ii], runtime.GOOS, runtime.GOARCH, runtime.GOROOT(), func(l *Loader) {
				l.fs = afero.NewCopyOnWriteFs(l.fs, overlayFs)
			})
			ls[ii] = l

			err := l.LoadDirectory(paths[ii])
			if err != nil {
				t.Fatalf("Error while loading: %s", err.Error())
			}

			l.Wait()
			wg.Done()
		}()
	}

	wg.Wait()

	errCount := 0
	for i := 0; i < 2; i++ {
		ls[i].Errors(func(file string, errs []FileError) {
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
		filepath.Join(rootPath, "foo"): 0,
		filepath.Join(rootPath, "bar"): 0,
		filepath.Join(rootPath, "baz"): 0,
	}
	le.caravan.Iter(func(_ collections.Hash, node *collections.Node) bool {
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
