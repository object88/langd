package simon

import "fmt"

type Simon struct{}

func (s *Simon) Says(phrase string) string {
	return fmt.Sprintf("SIMON SAYS: %s\n", phrase)
}
