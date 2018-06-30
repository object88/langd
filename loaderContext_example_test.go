package langd

import (
	"bytes"
	"fmt"
	"os"
	"runtime"
	"testing"

	"github.com/object88/langd/log"
)

var exampleIsAccessible = false

func init() {
	fi, err := os.Stat("../langd-example")
	if err != nil {
		return
	}

	if !fi.IsDir() {
		return
	}

	exampleIsAccessible = true
}

func Test_Scale(t *testing.T) {
	if !exampleIsAccessible {
		t.Skip("skipping because the `langd-example` project does not exist.")
	}

	path := "../langd-example"

	logger := log.Stdout()
	logger.SetLevel(log.Debug)
	le := NewLoaderEngine()
	defer le.Close()
	l := NewLoader(le, path, "darwin", "amd64", runtime.GOROOT())
	w := CreateWorkspace(le, logger)
	w.AssignLoader(l)

	err := l.LoadDirectory(path)
	if err != nil {
		t.Fatalf("Failed to load directory '%s':\n\t%s\n", path, err.Error())
	}

	fmt.Printf("Load directory started; blocking...\n")

	l.Wait()

	errCount := 0
	var buf bytes.Buffer
	l.Errors(func(file string, errs []FileError) {
		if len(errs) == 0 {
			return
		}
		buf.WriteString(fmt.Sprintf("Loading error in %s:\n", file))
		for k, err := range errs {
			buf.WriteString(fmt.Sprintf("\t%02d: %s:%d %s\n", k, err.Filename, err.Line, err.Message))
		}
		errCount += len(errs)
	})

	if errCount != 0 {
		buf.WriteString(fmt.Sprintf("Total: %d errors\n", errCount))
		t.Fatal(buf.String())
	}
}

func Benchmark_Scale(b *testing.B) {
	if !exampleIsAccessible {
		b.Skip("skipping because the `langd-example` project does not exist.")
	}

	logger := log.Stdout()
	logger.SetLevel(log.Debug)

	path := "../langd-example"

	for n := 0; n < b.N; n++ {
		le := NewLoaderEngine()
		defer le.Close()
		l := NewLoader(le, path, "darwin", "amd64", runtime.GOROOT())
		w := CreateWorkspace(le, logger)
		w.AssignLoader(l)

		l.LoadDirectory(path)
		l.Wait()
	}

}
