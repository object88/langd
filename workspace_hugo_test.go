package langd

import (
	"bytes"
	"fmt"
	"go/token"
	"testing"

	"github.com/object88/langd/log"
)

func Test_Workspace_Hugo(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping in short mode")
	}

	logger := log.Stdout()
	logger.SetLevel(log.Debug)
	l := NewLoader()
	w := CreateWorkspace(l, logger)

	done := l.Start()
	if done == nil {
		t.Fatal("Did not check channel back.\n")
	}
	fmt.Printf("Have done channel.\n")

	path := "/Users/bropa18/work/src/github.com/gohugoio/hugo"

	fmt.Printf("Going to load %s\n", path)

	err := l.LoadDirectory(path)
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

	p := &token.Position{
		Filename: "/Users/bropa18/work/src/github.com/gohugoio/hugo/helpers/processing_stats.go",
		Line:     109,
		Column:   24,
	}
	declPosition, err := w.LocateDeclaration(p)
	if err != nil {
		t.Error(err.Error())
	}
	if !declPosition.IsValid() {
		t.Error("Returned declaration position is not valid.")
	}
	if declPosition.Filename != "/Users/bropa18/work/src/github.com/gohugoio/hugo/vendor/github.com/olekukonko/tablewriter/table.go" {
		t.Errorf("Wrong file:\n\t%s\n", declPosition.Filename)
	}
	if declPosition.Line != 85 {
		t.Errorf("Wrong line:\n\t%d\n", declPosition.Line)
	}

	// t.Errorf("Dump logging...\n\t%s\n", declPosition.String())
}
