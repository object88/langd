package langd

import (
	"bytes"
	"fmt"
	"go/token"
	"io/ioutil"
	"os"
	"path/filepath"
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

	path := "../../gohugoio/hugo"

	logger := log.Stdout()
	logger.SetLevel(log.Debug)
	l := NewLoader()
	defer l.Close()
	lc := NewLoaderContext(l, path, runtime.GOOS, runtime.GOARCH, runtime.GOROOT())
	w := CreateWorkspace(l, logger)
	w.AssignLoaderContext(lc)

	err := l.LoadDirectory(lc, path)
	if err != nil {
		t.Fatalf("Failed to load directory '%s':\n\t%s\n", path, err.Error())
	}

	t.Log("Load directory started; blocking...\n")

	lc.Wait()

	t.Log("Load directory done\n")

	errCount := 0
	var buf bytes.Buffer
	l.Errors(lc, func(file string, errs []FileError) {
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

	csvPath, _ := filepath.Abs(filepath.Join(path, "vendor/github.com/olekukonko/tablewriter/csv.go"))
	tablePath, _ := filepath.Abs(filepath.Join(path, "vendor/github.com/olekukonko/tablewriter/table.go"))
	processingStatePath, _ := filepath.Abs(filepath.Join(path, "helpers/processing_stats.go"))

	byteContents, err := ioutil.ReadFile(csvPath)
	if err != nil {
		t.Fatal(err)
	}
	contents := string(byteContents)
	w.OpenFile(csvPath, contents)

	declPosition := &token.Position{Filename: tablePath, Line: 85, Column: 6}
	p := &token.Position{Filename: csvPath, Line: 33, Column: 7}

	testDeclaration(t, w, p, declPosition)

	testReferences(t, w, p, []*token.Position{
		declPosition,
		p,
		{Filename: processingStatePath, Line: 74, Column: 23},
		{Filename: processingStatePath, Line: 109, Column: 23},
	})
}
