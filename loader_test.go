package langd

import "testing"

func Test_Load(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping in short mode")
	}

	l := NewLoader()
	done := l.Start()

	l.LoadDirectory("./examples/foo")

	<-done

	errCount := 0
	l.Errors(func(file string, errs []FileError) {
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
