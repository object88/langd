package langd

import (
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"testing"
)

// Test_LoaderContext_Different_Root will require a different test context
// than FakeContext, which only allows one GOROOT.  We will want to start to
// use Afero (https://github.com/spf13/afero) in the Loader, so that we can
// provide memory-mapped complex file systems.
func Test_LoaderContext_Different_Root(t *testing.T) {
	cmd := exec.Command("which", "go")
	out, err := cmd.Output()
	if err != nil {
		t.Fatalf("Failed to locate the `go` executable\n\t%s\n", err.Error())
	}

	goAbsPath, err := filepath.Abs(strings.TrimSpace(string(out)))
	if err != nil {
		t.Fatalf("Failed to get absolute path for `go`\n\t%s\n", err.Error())
	}

	goPath, err := filepath.EvalSymlinks(goAbsPath)
	if err != nil {
		t.Fatalf("Failed to evalulate symlinks for %s\n\t%s\n", goAbsPath, err.Error())
	}

	goRoot, _ := filepath.Split(filepath.Dir(goPath))
	fmt.Printf("go path: %s\n", goPath)
	fmt.Printf("go root: %s\n", goRoot)

	le := NewLoaderEngine()
	defer le.Close()

	errCount := 0
	path := "../langd-example"
	fn := func(file string, errs []FileError) {
		if len(errs) == 0 {
			return
		}
		t.Errorf("Loading error in %s:\n", file)
		for k, err := range errs {
			t.Errorf("\t%d: [%d:%d] %s\n", k, err.Line, err.Column, err.Message)
		}
		errCount += len(errs)
	}

	var wg sync.WaitGroup
	wg.Add(2)

	procfn := func(root string) {
		l := NewLoader(le, "something", "darwin", "amd64", root)

		fmt.Printf("Starting at goroot %s...\n", l.context.GOROOT)

		err = l.LoadDirectory(path)
		if err != nil {
			t.Fatalf("Error while loading: %s", err.Error())
		}
		l.Wait()
		l.Errors(fn)

		wg.Done()
	}

	procfn(goRoot)
	procfn("/usr/local/Cellar/go/1.9.4/libexec/")

	wg.Wait()

	if errCount != 0 {
		t.Fatalf("Found %d errors", errCount)
	}

	t.Error("Not implemented: final check of loaded ASTs")
}
