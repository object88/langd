package langd

import (
	"sync"
	"testing"
	"time"

	"github.com/object88/langd/collections"
	"golang.org/x/tools/go/buildutil"
)

func Test_LoadContext_Same_Package_Same_Env(t *testing.T) {
	src := `package bar
	var BarVal int = 0`

	packages := map[string]map[string]string{
		"bar": map[string]string{
			"bar.go": src,
		},
	}

	loader := NewLoader()
	defer loader.Close()
	// loader.Log.SetLevel(log.Debug)
	// done := loader.Start()

	lc1 := NewLoaderContext(loader, "/go/src/bar", "darwin", "x86", "/go", func(lc LoaderContext) {
		lc.(*loaderContext).context = buildutil.FakeContext(packages)
	})

	lc2 := NewLoaderContext(loader, "/go/src/bar", "linux", "arm", "/go", func(lc LoaderContext) {
		lc.(*loaderContext).context = buildutil.FakeContext(packages)
	})

	err := loader.LoadDirectory(lc1, "/go/src/bar")
	if err != nil {
		t.Fatalf("(1) Error while loading: %s", err.Error())
	}

	err = loader.LoadDirectory(lc2, "/go/src/bar")
	if err != nil {
		t.Fatalf("(2) Error while loading: %s", err.Error())
	}

	// <-done
	lc1.Wait()
	lc2.Wait()

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

	packageCount := 0
	loader.Caravan().Iter(func(key collections.Key, node *collections.Node) bool {
		packageCount++
		return true
	})

	if packageCount != 2 {
		t.Errorf("Expected to find 2 packages; found %d\n", packageCount)
	}
}

func Test_LoadContext_Same_Package_Different_Env(t *testing.T) {
	src := `package bar
	var BarVal BarType = 0`

	srcDarwin := `package bar
	type BarType int32`

	srcLinux := `package bar
	type BarType int64`

	packages := map[string]map[string]string{
		"bar": map[string]string{
			"bar.go":        src,
			"bar_darwin.go": srcDarwin,
			"bar_linux.go":  srcLinux,
		},
	}

	loader := NewLoader()
	defer loader.Close()
	// loader.Log.SetLevel(log.Debug)
	// done := loader.Start()

	envs := [][]string{
		[]string{"darwin", "amd"},
		[]string{"linux", "arm"},
	}

	wg := sync.WaitGroup{}
	wg.Add(2)

	for i := 0; i < 2; i++ {
		env := envs[i]
		go func() {
			lc := NewLoaderContext(loader, "/go/src/bar", env[0], env[1], "/go", func(lc LoaderContext) {
				lc.(*loaderContext).context = buildutil.FakeContext(packages)
			})

			err := loader.LoadDirectory(lc, "/go/src/bar")
			if err != nil {
				t.Fatalf("Error while loading: %s", err.Error())
			}

			lc.Wait()

			time.Sleep(100 * time.Millisecond)

			wg.Done()
		}()
	}

	// <-done
	wg.Wait()

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

	packageCount := 0
	loader.Caravan().Iter(func(key collections.Key, node *collections.Node) bool {
		packageCount++
		p := node.Element.(*Package)
		if len(p.files) != 2 {
			t.Errorf("Package '%s' has the wrong number of files; expected 2, got %d", p.shortPath, len(p.files))
		}
		return true
	})

	if packageCount != 2 {
		t.Errorf("Expected to find 2 packages; found %d\n", packageCount)
	}
}
