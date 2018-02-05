package langd

import (
	"os"
	"testing"

	"github.com/object88/langd/log"
	"golang.org/x/tools/go/buildutil"
)

const identImportsTestProgram1 = `package foo

import (
	"../bar"
)

func add1(param1 int) int {
	bar.CountCall("add1")

	if bar.TotalCalls == 100 {
		// That's a lot of calls...
		return param1
	}

	param1++
	return param1
}
`

const identImportsTestProgram2 = `package bar

var calls = map[string]int{}
var TotalCalls = 0

func CountCall(source string) {
	TotalCalls++

	call, ok := calls[source]
	if !ok {
		call = 1
	} else {
		call++
	}
	calls[source] = call
}
`

// func Test_FromImports_LocateReferences_AsFunc(t *testing.T) {
// 	w := setup2(t)

// 	callCountInvokeOffset := nthIndex(identImportsTestProgram1, "CountCall", 0)
// 	callCountInvokePosition := w.Loader.Fset.Position(token.Pos(callCountInvokeOffset + 1))
// 	callCountIdent, _ := w.LocateIdent(&callCountInvokePosition)

// 	refPositions := w.LocateReferences(callCountIdent)
// 	if nil == refPositions {
// 		t.Fatalf("Returned nil from LocateReferences")
// 	}
// }

func setup2(t *testing.T) *Workspace {
	packages := map[string]map[string]string{
		"foo": map[string]string{
			"foo.go": identImportsTestProgram1,
		},
		"bar": map[string]string{
			"bar.go": identImportsTestProgram2,
		},
	}

	fc := buildutil.FakeContext(packages)
	loader := NewLoader(func(l *Loader) {
		l.context = fc
	})
	w := CreateWorkspace(loader, log.CreateLog(os.Stdout))
	w.log.SetLevel(log.Verbose)

	done := loader.Start()
	loader.LoadDirectory("/go/src/foo")
	<-done

	errCount := 0
	w.Loader.Errors(func(file string, errs []FileError) {
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

	return w
}
