package langd

import (
	"bytes"
	"fmt"
	"go/token"
	"os"
	"runtime"
	"testing"

	"github.com/object88/langd/log"
)

var hugoIsAccessible = false

func init() {
	fi, err := os.Stat("../../gohugoio/hugo")
	if err != nil {
		return
	}

	if !fi.IsDir() {
		return
	}

	hugoIsAccessible = true
}

func Test_Workspace_Hugo(t *testing.T) {
	if testing.Short() || !hugoIsAccessible {
		t.Skip("skipping in short mode")
	}

	logger := log.Stdout()
	logger.SetLevel(log.Debug)
	l := NewLoader()
	lc := NewLoaderContext(l, runtime.GOOS, runtime.GOARCH)
	w := CreateWorkspace(l, lc, logger)

	done := l.Start()
	if done == nil {
		t.Fatal("Did not check channel back.\n")
	}

	path := "../../gohugoio/hugo"
	err := l.LoadDirectory(lc, path)
	if err != nil {
		t.Fatalf("Failed to load directory '%s':\n\t%s\n", path, err.Error())
	}

	fmt.Printf("Load directory started; blocking...\n")

	<-done

	fmt.Printf("Load directory done\n")

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

	declPosition := &token.Position{
		Filename: "/Users/bropa18/work/src/github.com/gohugoio/hugo/vendor/github.com/olekukonko/tablewriter/table.go",
		Line:     85,
		Column:   6,
	}

	p := &token.Position{
		Filename: "/Users/bropa18/work/src/github.com/gohugoio/hugo/vendor/github.com/olekukonko/tablewriter/csv.go",
		Line:     33,
		Column:   7,
	}

	testDeclaration(t, w, p, declPosition)

	testReferences(t, w, p, []*token.Position{
		declPosition,
		p,
		{
			Filename: "/Users/bropa18/work/src/github.com/gohugoio/hugo/helpers/processing_stats.go",
			Line:     74,
			Column:   23,
		},
		{
			Filename: "/Users/bropa18/work/src/github.com/gohugoio/hugo/helpers/processing_stats.go",
			Line:     109,
			Column:   23,
		},
	})
}
