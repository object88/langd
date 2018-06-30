package langd

import (
	"path/filepath"
	"testing"
)

func Test_Workspace_Change_Creates_Error(t *testing.T) {
	src1 := `package foo
	var Foof int = 1`

	src2 := `package bar
	import "../foo"
	func IncFoof() int {
		foo.Foof++
		return foo.Foof
	}`

	rootPath, overlayFs := createOverlay(map[string]string{
		filepath.Join("foo", "foo.go"): src1,
		filepath.Join("bar", "bar.go"): src2,
	})
	barPath := filepath.Join(rootPath, "bar")
	fooGoPath := filepath.Join(rootPath, "foo", "foo.go")

	w, closer := workspaceSetup(t, barPath, overlayFs, false)
	defer closer()

	w.OpenFile(fooGoPath, src1)
	w.Loader.Wait()

	w.ChangeFile(fooGoPath, 1, 5, 1, 9, "FOOF")
	w.Loader.Wait()

	errCount := 0
	w.Loader.Errors(func(file string, errs []FileError) {
		errCount += len(errs)
	})

	if errCount == 0 {
		t.Errorf("Did not get any errors")
	}
}

func Test_Workspace_Change_Creates_Error_Indirect(t *testing.T) {
	src1 := `package foo
	type Foo struct {}
	func (f *Foo) Add() int {
		return 0
	}`

	src2 := `package bar
	import "../foo"
	type Bar struct {
		F *foo.Foo
	}`

	src3 := `package baz
	import "../bar"
	func Do(b *bar.Bar) int {
		return b.F.Add()
	}`

	rootPath, overlayFs := createOverlay(map[string]string{
		filepath.Join("foo", "foo.go"): src1,
		filepath.Join("bar", "bar.go"): src2,
		filepath.Join("baz", "baz.go"): src3,
	})
	bazPath := filepath.Join(rootPath, "baz")
	fooGoPath := filepath.Join(rootPath, "foo", "foo.go")

	w, closer := workspaceSetup(t, bazPath, overlayFs, false)
	defer closer()

	w.OpenFile(fooGoPath, src1)
	w.Loader.Wait()

	w.ChangeFile(fooGoPath, 2, 15, 2, 18, "Inc")
	w.Loader.Wait()

	errCount := 0
	w.Loader.Errors(func(file string, errs []FileError) {
		errCount += len(errs)
	})

	if errCount == 0 {
		t.Errorf("Did not get any errors")
	}
}
