package langd

import (
	"fmt"
)

func CalculateOffsetForPosition(bytes []byte, line, character int) (int, error) {
	c := 0
	l := 0
	offset := 0

	for _, b := range bytes {
		if l == line && c == character {
			return offset, nil
		}

		offset++
		if b == '\n' {
			l++
			c = -1
		}
		c++
	}

	return 0, fmt.Errorf("Failed to get offset for position [%d,%d]", line, character)
}
