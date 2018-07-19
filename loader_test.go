package langd

import (
	"path/filepath"
	"runtime"
	"sync"
	"testing"

	"github.com/object88/langd/collections"
	"github.com/spf13/afero"
)

func Test_LoadContext_Same_Package_Same_Env(t *testing.T) {
	src := `package bar
	var BarVal int = 0`

	rootPath, overlayFs := createOverlay(map[string]string{
		filepath.Join("bar", "bar.go"): src,
	})
	barPath := filepath.Join(rootPath, "bar")

	le := NewLoaderEngine()
	defer le.Close()

	l1 := NewLoader(le, "darwin", "x86", runtime.GOROOT(), func(l *Loader) {
		l.fs = afero.NewCopyOnWriteFs(l.fs, overlayFs)
	})

	l2 := NewLoader(le, "linux", "arm", runtime.GOROOT(), func(l *Loader) {
		l.fs = afero.NewCopyOnWriteFs(l.fs, overlayFs)
	})

	err := l1.LoadDirectory(barPath)
	if err != nil {
		t.Fatalf("(1) Error while loading: %s", err.Error())
	}

	err = l2.LoadDirectory(barPath)
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
		_, err := l.FindDistinctPackage(barPath)
		if err != nil {
			failed = true
			t.Errorf("Failed to find package '%s'", barPath)
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

	rootPath, overlayFs := createOverlay(map[string]string{
		filepath.Join("bar", "bar.go"):        src,
		filepath.Join("bar", "bar_darwin.go"): srcDarwin,
		filepath.Join("bar", "bar_linux.go"):  srcLinux,
	})
	barPath := filepath.Join(rootPath, "bar")

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
			l := NewLoader(le, env[0], env[1], runtime.GOROOT(), func(l *Loader) {
				l.fs = afero.NewCopyOnWriteFs(l.fs, overlayFs)
			})
			ls[ii] = l

			err := l.LoadDirectory(barPath)
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
		_, err := l.FindDistinctPackage(barPath)
		if err != nil {
			t.Errorf("Failed to find package '%s'", barPath)
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
