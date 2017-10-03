package langd

import (
	"fmt"
	"go/token"
	"io"
	"unicode/utf8"
)

// OverrunError is returned from CalculateOffsetForPosistion when the
// requested line does not have enough characters to find an offset
type OverrunError struct {
	otype OverrunType
	line  int
	char  int
}

// OverrunType describes the reason for the overrun error
type OverrunType int

const (
	// Line indicates that the overrun occured because there were not enough
	// lines to get the offset for the requested position
	Line OverrunType = iota

	// Character indicates that the overrun occured because the line did not
	// have enough characters to get the offset for the requested position
	Character
)

// NewOverrunError creates a new OverrunError instance
func NewOverrunError(overrun OverrunType, line, char int) *OverrunError {
	return &OverrunError{overrun, line, char}
}

func (oe *OverrunError) Error() string {
	if oe.otype == Line {
		return fmt.Sprintf(
			"Insufficient lines to locate line %d, character %d\n",
			oe.line,
			oe.char)
	}
	return fmt.Sprintf(
		"Insufficient characters on line %d for character %d\n",
		oe.line,
		oe.char)
}

// CalculateOffsetForPosition scans a rope to get to the rune offset at the
// given line and character
func CalculateOffsetForPosition(read io.Reader, line, character int) (int, error) {
	if line == 0 && character == 0 {
		return 0, nil
	}

	c := 0
	l := 0
	offset := 0

	bytes := make([]byte, 1024)
	for {
		n, err := read.Read(bytes)
		if err != nil {
			if err == io.EOF {
				return 0, NewOverrunError(Line, line, character)
			}
			return 0, err
		}
		if n == 0 {
			ot := Line
			if l == line {
				ot = Character
			}
			return 0, NewOverrunError(ot, line, character)
		}

		o := 0
		for o < n {
			r, s := utf8.DecodeRune(bytes[o:n])
			o += s
			offset += s
			if r == '\n' {
				l++
				c = -1
				if l > line {
					return 0, NewOverrunError(Character, line, character)
				}
			}
			c++

			if l == line && c == character {
				return offset, nil
			}
		}
	}
}

func WithinPosition(target, start, end *token.Position) bool {
	if target.Line < start.Line || target.Line > end.Line {
		return false
	}

	if target.Line == start.Line && target.Column < start.Column {
		return false
	}

	if target.Line == end.Line && target.Column >= end.Column {
		return false
	}

	return true
}
