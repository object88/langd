package langd

import (
	"fmt"
	"unicode/utf8"

	"github.com/object88/rope"
)

// CalculateOffsetForPosition scans a rope to get to the rune offset at the
// given line and character
func CalculateOffsetForPosition(r *rope.Rope, line, character int) (int, error) {
	c := 0
	l := 0
	offset := 0

	bytes := make([]byte, 1024)
	read := r.NewReader()
	for {
		n, err := read.Read(bytes)
		if err != nil {
			return 0, err
		}
		if n == 0 {
			return 0, fmt.Errorf("Failed to get offset for position [%d,%d]", line, character)
		}

		o := 0
		subbytes := bytes[o:n]
		for o < n {
			if l == line && c == character {
				return offset, nil
			}

			r0, s := utf8.DecodeRune(subbytes[o:n])
			o += s
			offset += s
			if r0 == '\n' {
				l++
				c = -1
			}
			c++
		}
	}
}
