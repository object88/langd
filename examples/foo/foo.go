package foo

import (
	"math"
)

type Foo struct {
	X int
	Y int
}

func NewFoo(x, y int) *Foo {
	return &Foo{
		X: x,
		Y: y,
	}
}

func (f *Foo) Move(d int, dir int) bool {
	if dir < 0 || dir > 359 {
		return false
	}

	fdir := float64(dir)
	fd := float64(d)

	sin := math.Sin(fdir)
	fdeltaX := fd * sin
	deltaX := int(fdeltaX)
	a := (fdeltaX - float64(deltaX)) * 10
	if a > 5 {
		deltaX++
	}
	f.X += deltaX

	cos := math.Cos(fdir)
	fdeltaY := fd * cos
	deltaY := int(fdeltaY)
	b := (fdeltaY - float64(deltaY)) * 10
	if b > 5 {
		deltaY++
	}
	f.Y += deltaY

	return true
}
