package langd

// workspace_test.go does not contain actual tests, but rather utility
// functions that support testing workspace methods.

import (
	"bytes"
	"fmt"
	"go/token"
	"os"
	"runtime"
	"testing"

	"github.com/object88/langd/log"
	"golang.org/x/tools/go/buildutil"
)

func workspaceSetup(t *testing.T, startingPath string, packages map[string]map[string]string, expectFailure bool) (*Workspace, *Loader, func()) {
	fc := buildutil.FakeContext(packages)
	le := NewLoaderEngine()
	l := NewLoader(le, startingPath, runtime.GOOS, runtime.GOARCH, "/go", func(l *Loader) {
		l.context = fc
	})
	w := CreateWorkspace(le, log.CreateLog(os.Stdout))
	w.AssignLoader(l)
	w.log.SetLevel(log.Verbose)

	t.Logf("About to load directory '%s'\n", startingPath)
	err := l.LoadDirectory(startingPath)
	if err != nil {
		t.Fatalf("Error while loading directory '%s': %s", startingPath, err.Error())
	}
	t.Logf("Finished loading directory\n")

	t.Logf("Waiting for complete\n")
	l.Wait()
	t.Logf("Complete\n")

	if expectFailure {
		errCount := 0
		w.Loader.Errors(func(file string, errs []FileError) {
			errCount += len(errs)
		})
		if errCount == 0 {
			t.Fatal("Expected errors, but got none\n")
		}
	} else {
		errCount := 0
		var buf bytes.Buffer
		w.Loader.Errors(func(file string, errs []FileError) {
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

	return w, l, func() { le.Close() }
}

func testDeclaration(t *testing.T, w *Workspace, usagePosition, expectedDeclPosition *token.Position) {
	declPosition, err := w.LocateDeclaration(usagePosition)
	if err != nil {
		t.Fatal(err.Error())
	}

	testPosition(t, declPosition, expectedDeclPosition)
}

func testPosition(t *testing.T, actual, expected *token.Position) {
	if actual == nil {
		t.Fatalf("actual is nil")
	}

	if !actual.IsValid() {
		t.Fatalf("Returned position '%s' is not valid", actual.String())
	}

	if actual.Filename != expected.Filename {
		t.Fatalf("Incorrect filename: got '%s', expected '%s'", actual.Filename, expected.Filename)
	}

	if actual.Line != expected.Line {
		t.Fatalf("Incorrect line: got %d, expected %d", actual.Line, expected.Line)
	}

	if actual.Column != expected.Column {
		t.Fatalf("Incorrect column: got %d, expected %d", actual.Column, expected.Column)
	}
}

func comparePosition(actual, expected *token.Position) bool {
	if actual == nil {
		return false
	}

	if !actual.IsValid() {
		return false
	}

	if actual.Filename != expected.Filename {
		return false
	}

	if actual.Line != expected.Line {
		return false
	}

	if actual.Column != expected.Column {
		return false
	}

	return true
}
