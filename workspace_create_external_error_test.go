package langd

import "testing"

func Test_Workspace_Change_Creates_Error(t *testing.T) {
	src1 := `package foo
	var Foof int = 1`

	src2 := `package bar
	import "../foo"
	func IncFoof() int {
		foo.Foof++
		return foo.Foof
	}`

	packages := map[string]map[string]string{
		"foo": map[string]string{
			"foo.go": src1,
		},
		"bar": map[string]string{
			"bar.go": src2,
		},
	}

	w := workspaceSetup(t, "/go/src/bar", packages, false)

	done := w.Loader.Start()

	w.OpenFile("/go/src/foo/foo.go", src1)
	<-done

	w.ChangeFile("/go/src/foo/foo.go", 1, 5, 1, 9, "FOOF")
	<-done

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

	packages := map[string]map[string]string{
		"foo": map[string]string{
			"foo.go": src1,
		},
		"bar": map[string]string{
			"bar.go": src2,
		},
		"baz": map[string]string{
			"baz.go": src3,
		},
	}

	w := workspaceSetup(t, "/go/src/baz", packages, false)

	done := w.Loader.Start()

	w.OpenFile("/go/src/foo/foo.go", src1)
	<-done

	w.ChangeFile("/go/src/foo/foo.go", 2, 15, 2, 18, "Inc")
	<-done

	errCount := 0
	w.Loader.Errors(func(file string, errs []FileError) {
		errCount += len(errs)
	})

	if errCount == 0 {
		t.Errorf("Did not get any errors")
	}
}