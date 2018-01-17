package langd

import (
	"testing"

	"golang.org/x/tools/go/buildutil"
)

const differentDirImportsTestProgram1 = `package foo

import (
	"../gobar"
)

func add1(param1 int) int {
	bar.TotalCalls++

	return param1+1
}
`

const differentDirImportsTestProgram2 = `package bar

var TotalCalls = 0
`

func Test_Load_PackageWithDifferentDir(t *testing.T) {
	packages := map[string]map[string]string{
		"foo": map[string]string{
			"foo.go": differentDirImportsTestProgram1,
		},
		"gobar": map[string]string{
			"stats.go": differentDirImportsTestProgram2,
		},
	}

	fc := buildutil.FakeContext(packages)
	loader := NewLoader(func(l *Loader) {
		l.context = fc
	})
	done := loader.Start()
	loader.LoadDirectory("/go/src/foo")
	<-done

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
}
