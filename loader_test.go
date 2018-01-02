package langd

import "testing"

func Test_Load(t *testing.T) {
	l := NewLoader()
	done := l.Start()

	l.LoadDirectory("./examples/foo")

	<-done
}
