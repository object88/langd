package langd

import (
	"bytes"
	"fmt"
	"strings"
	"testing"
)

const testProgram = `package foo

var foo = 13
var bar = 31
`

func Test_Position(t *testing.T) {
	r := strings.NewReader(testProgram)

	n, err := CalculateOffsetForPosition(r, 0, 4)
	if err != nil {
		t.Errorf("Unexpected error: %s", err.Error())
	}
	if n != 4 {
		t.Errorf("Incorrect position returned: %d", n)
	}
}

func Test_Position_LastRowAndColumn(t *testing.T) {
	r := strings.NewReader(testProgram)

	n, err := CalculateOffsetForPosition(r, 4, 0)
	if err != nil {
		t.Errorf("Unexpected error: %s", err.Error())
	}
	if n != 39 {
		t.Errorf("Incorrect position returned: %d", n)
	}
}

func Test_Position_Overrun_Column(t *testing.T) {
	r := strings.NewReader(testProgram)

	n, err := CalculateOffsetForPosition(r, 0, 40)
	if err == nil {
		t.Error("Expected error")
	}
	if _, ok := err.(*OverrunError); !ok {
		t.Errorf("Returned error is incorrect type; got '%s'", err.Error())
	}
	if n != 0 {
		t.Errorf("Incorrect position returned: %d", n)
	}
}

func Test_Position_Overrun_Line(t *testing.T) {
	r := strings.NewReader(testProgram)

	n, err := CalculateOffsetForPosition(r, 6, 1)
	if err == nil {
		t.Error("Expected error")
	}
	if _, ok := err.(*OverrunError); !ok {
		t.Errorf("Returned error is incorrect type; got '%s'", err.Error())
	}
	if n != 0 {
		t.Errorf("Incorrect position returned: %d", n)
	}
}

func Test_Position_Multiple_Reads(t *testing.T) {
	// Create a "file" with more than 1k bytes
	var buf bytes.Buffer
	for i := 0; i < 20; i++ {
		// Each line is 15 bytes
		buf.WriteString(fmt.Sprintf("%02d: 0123456789\n", i))
	}
	s := buf.String()

	r := strings.NewReader(s)

	// (18 * 15) + 6 = 276
	expected := 276
	n, err := CalculateOffsetForPosition(r, 18, 6)
	if err != nil {
		t.Errorf("Unexpected error: %s", err.Error())
	}
	if n != expected {
		t.Errorf("Incorrect offset calculated; expected %d, got %d", expected, n)
	}
}
