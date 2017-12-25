package repeater

import "github.com/object88/langd/examples/echo/simon"

// Repeat repeats a message back, as if Simon said it
func Repeat(s string) string {
	// s = fmt.Sprintf("SIMON SAYS: %s", s)
	simon := &simon.Simon{}
	s = simon.Says(s)
	return s
}
