package langd

import (
	"fmt"
	"go/token"
	"os"
	"strings"
	"testing"

	"github.com/object88/langd/log"
	"golang.org/x/tools/go/buildutil"
)

func Test_LocateIdent(t *testing.T) {
	const testProgram = `package foo

func add1(add1Param1 int) int {
	add1result := add1Param1
	add1result++
	return add1result
}

func addWhilePos(addWhilePosParam1, addWhilePosParam2 int) int {
	if addWhilePosParam2 == 0 {
		return addWhilePosParam1
	}
	return addWhilePos(addWhilePosParam1 + 1, addWhilePosParam2 - 1)
}
`

	packages := map[string]map[string]string{
		"foo": map[string]string{
			"foo.go": testProgram,
		},
	}

	fc := buildutil.FakeContext(packages)
	loader := NewLoader(func(l *Loader) {
		l.context = fc
	})
	w := CreateWorkspace(loader, log.CreateLog(os.Stdout))

	done := loader.Start()
	loader.LoadDirectory("/go/src/foo")
	<-done

	w.AssignAST()

	errCount := 0
	w.Loader.Errors(func(file string, errs []FileError) {
		errCount++
	})

	if errCount != 0 {
		t.Fatalf("Found %d errors", errCount)
	}

	identName := "add1result"
	offset := nthIndex(testProgram, identName, 0) + 1

	// Find an ident a couple of characters into the word
	p := w.Fset.Position(token.Pos(offset + 2))
	fmt.Printf("p: %#v\n", p)
	ident, err := w.LocateIdent(&p)
	if err != nil {
		t.Errorf("Got error: %s", err.Error())
	}

	if ident == nil {
		t.Errorf("Did not get ident back")
	}
	if int(ident.Pos()) != offset {
		t.Errorf("Ident is at wrong position: got %d; expected %d", ident.Pos(), offset)
	}
	if ident.Name != identName {
		t.Errorf("Ident has wrong name; got '%s'; expected '%s'", ident.Name, identName)
	}
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
