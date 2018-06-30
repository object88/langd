package langd

import (
	"sync"
	"testing"

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

	le := NewLoaderEngine()
	defer le.Close()

	l1 := NewLoader(le, "/go/src/bar", "darwin", "x86", "/go", func(l *Loader) {
		l.context = buildutil.FakeContext(packages)
	})

	l2 := NewLoader(le, "/go/src/bar", "linux", "arm", "/go", func(l *Loader) {
		l.context = buildutil.FakeContext(packages)
	})

	err := l1.LoadDirectory("/go/src/bar")
	if err != nil {
		t.Fatalf("(1) Error while loading: %s", err.Error())
	}

	err = l2.LoadDirectory("/go/src/bar")
	if err != nil {
		t.Fatalf("(2) Error while loading: %s", err.Error())
	}

	l1.Wait()
	l2.Wait()

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

	l1.Errors(fn)
	l2.Errors(fn)

	if errCount != 0 {
		t.Fatalf("Found %d errors", errCount)
	}

	packageCount := 0
	le.caravan.Iter(func(hash collections.Hash, node *collections.Node) bool {
		packageCount++
		return true
	})

	if packageCount != 2 {
		t.Errorf("Expected to find 1 package; found %d\n", packageCount)
	}

	failed := false
	for _, l := range []*Loader{l1, l2} {
		_, err := l.FindDistinctPackage("/go/src/bar")
		if err != nil {
			failed = true
			t.Errorf("Failed to find package '%s'", "/go/src/bar")
		}
	}

	if failed {
		le.caravan.Iter(func(hash collections.Hash, n *collections.Node) bool {
			t.Errorf("Have hash 0x%x: %s\n", hash, n.Element.(*DistinctPackage))
			return true
		})
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

	le := NewLoaderEngine()
	defer le.Close()

	envs := [][]string{
		[]string{"darwin", "amd"},
		[]string{"linux", "arm"},
	}

	wg := sync.WaitGroup{}
	wg.Add(2)

	ls := make([]*Loader, 2)

	for i := 0; i < 2; i++ {
		ii := i
		env := envs[i]
		go func() {
			l := NewLoader(le, "/go/src/bar", env[0], env[1], "/go", func(l *Loader) {
				l.context = buildutil.FakeContext(packages)
			})
			ls[ii] = l

			err := l.LoadDirectory("/go/src/bar")
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

	packageCount := 0
	le.caravan.Iter(func(hash collections.Hash, node *collections.Node) bool {
		packageCount++
		return true
	})

	if packageCount != 2 {
		t.Errorf("Expected to find 1 packages; found %d\n", packageCount)
	}

	for _, l := range ls {
		_, err := l.FindDistinctPackage("/go/src/bar")
		if err != nil {
			t.Errorf("Failed to find package '%s'", "/go/src/bar")
		}
	}
	// dp := n.Element.(*DistinctPackage)

	// if 2 != len(p.distincts) {
	// 	t.Errorf("Expected to find 2 distinct packages; found %d", len(p.distincts))
	// }

	// for _, dp := range p.distincts {
	// 	if len(dp.files) != 2 {
	// 		t.Errorf("Package %s has the wrong number of files; expected 2, got %d", dp, len(dp.files))
	// 	}
	// }

	// for i := 0; i < 2; i++ {
	// 	le.caravan.Iter(func(hash collections.Hash, node *collections.Node) bool {
	// 		p := node.Element.(*Package)
	// 		dp := p.distincts[lcs[i].GetDistinctHash()]
	// 		if len(dp.files) != 2 {
	// 			t.Errorf("Package '%s' has the wrong number of files; expected 2, got %d", p.shortPath, len(dp.files))
	// 		}
	// 		return true
	// 	})
	// }
}
