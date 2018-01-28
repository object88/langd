package langd

import (
	"fmt"
	"go/token"
	"os"
	"strings"
	"sync"
	"testing"

	"github.com/object88/langd/log"
	"golang.org/x/tools/go/buildutil"
)

const identTestProgram = `package foo

var calls = map[string]int{}
var totalCalls = 0

func add1(add1Param1 int) int {
	countCall("add1")

	add1result := add1Param1
	add1result++
	return add1result
}

func countCall(source string) {
	totalCalls++

	call, ok := calls[source]
	if !ok {
		call = 1
	} else {
		call++
	}
	calls[source] = call
}

func addWhilePos(addWhilePosParam1, addWhilePosParam2 int) int {
	if addWhilePosParam2 == 0 {
		return addWhilePosParam1
	}
	return addWhilePos(addWhilePosParam1 + 1, addWhilePosParam2 - 1)
}
`

func Test_LocateIdent_OnIdent(t *testing.T) {
	identName := "add1result"
	w := setup(t)

	tests := []struct {
		name   string
		offset int
	}{
		{
			name:   "AtStart",
			offset: 0,
		},
		{
			name:   "Within",
			offset: 2,
		},
		{
			name:   "AtEnd",
			offset: len(identName) - 1,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			offset := nthIndex(identTestProgram, identName, 0)
			pos := offset + 1

			p := w.Loader.Fset.Position(token.Pos(pos + tc.offset))
			ident, err := w.LocateIdent(&p)
			if err != nil {
				t.Errorf("Got error: %s", err.Error())
			}
			if ident == nil {
				t.Errorf("Did not get ident back")
			}
			if int(ident.Pos()) != pos {
				t.Errorf("Ident is at wrong position: got %d; expected %d", ident.Pos(), pos)
			}
			if ident.Name != identName {
				t.Errorf("Ident has wrong name; got '%s'; expected '%s'", ident.Name, identName)
			}
		})
	}
}

func Test_LocateIdent_OnKeyword(t *testing.T) {
	identName := "func"
	w := setup(t)

	offset := nthIndex(identTestProgram, identName, 0)
	pos := offset + 1

	p := w.Loader.Fset.Position(token.Pos(pos))
	ident, err := w.LocateIdent(&p)
	if err != nil {
		t.Errorf("Got error: %s", err.Error())
	}
	if ident != nil {
		t.Errorf("Got ident back at %d", ident.Pos())
	}
}

func Test_LocateReferences(t *testing.T) {
	w := setup(t)

	identName := "add1result"
	offset := nthIndex(identTestProgram, identName, 0)

	// Find an ident a couple of characters into the word
	// Must add 1, then nudging in 2 characters.
	p := w.Loader.Fset.Position(token.Pos(offset + 3))
	fmt.Printf("p: %#v\n", p)
	ident, _ := w.LocateIdent(&p)
	if ident == nil {
		t.Fatalf("Did not get ident back")
	}

	refPositions := w.LocateReferences(ident)
	if nil == refPositions {
		t.Errorf("Did not get any references back")
	}

	expectedOffsets := []int{
		nthIndex(identTestProgram, identName, 1),
		nthIndex(identTestProgram, identName, 2),
	}

	for _, v := range *refPositions {
		found := false
		for _, v2 := range expectedOffsets {
			if v.Offset == v2 {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Got refPosition %d is not among expected offsets", v.Offset)
		}
	}
}

var _o sync.Once
var _w *Workspace

func setup(t *testing.T) *Workspace {
	_o.Do(func() {
		packages := map[string]map[string]string{
			"foo": map[string]string{
				"foo.go": identTestProgram,
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

		w.AssignAST()

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

		_w = w
	})

	return _w
}

func Test_nthIndex(t *testing.T) {
	//    0123456789
	s := "abcdefghijabcdefghijabcdefghij"
	x := nthIndex(s, "x", 0)
	if -1 != x {
		t.Errorf("Failed to return -1 for absent substring; got %d.", x)
	}

	x = nthIndex(s, "abc", 0)
	if 0 != x {
		t.Errorf("Failed to return 0 for first instance of substring; got %d.", x)
	}

	x = nthIndex(s, "abc", 1)
	if 10 != x {
		t.Errorf("Failed to return 10 for second instance of substring; got %d.", x)
	}

	x = nthIndex(s, "abc", 2)
	if 20 != x {
		t.Errorf("Failed to return 20 for third instance of substring; got %d.", x)
	}
}

func nthIndex(s string, sub string, n int) int {
	offset := 0
	for i := 0; i < n; i++ {
		loc := strings.Index(s, sub)
		if loc == -1 {
			return loc
		}
		offset += loc + 1
		s = s[loc+1:]
	}
	return offset + strings.Index(s, sub)
}

func nthIndex2(w *Workspace, path, s, sub string, n int) int {
	astFile, ok := w.Files[path]
	if !ok {
		panic(fmt.Sprintf("No ast.File at %s\n", path))
	}
	w.Loader.Fset.Position(astFile.Pos())

	offset := 0
	for i := 0; i < n; i++ {
		loc := strings.Index(s, sub)
		if loc == -1 {
			return loc
		}
		offset += loc + 1
		s = s[loc+1:]
	}
	return offset + strings.Index(s, sub)
}
