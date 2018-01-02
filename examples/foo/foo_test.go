package foo

import "testing"

func Test_GoWest(t *testing.T) {
	f := NewFoo(0, 0)
	b := f.Move(2, 90)
	if !b {
		t.Error("Got err")
	}
	if f.X != 2.0 {
		t.Error("Bad X")
	}
	if f.Y != 0 {
		t.Error("Bad Y")
	}
}

// import "runtime"

// func GoWest() string {
// 	f := NewFoo(0, 0)
// 	b := f.Move(2, 90)
// 	if !b {
// 		return runtime.GOROOT() + " Got err"
// 	}
// 	if f.X != 2.0 {
// 		return runtime.GOROOT() + " Bad X"
// 	}
// 	if f.Y != 0 {
// 		return runtime.GOROOT() + " Bad Y"
// 	}

// 	return ""
// }
