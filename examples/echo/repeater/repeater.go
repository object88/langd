package repeater

import "fmt"

// Repeat repeats a message back, as if Simon said it
func Repeat(s string) string {
	s = fmt.Sprintf("SIMON SAYS: %s", s)
	return s
}
