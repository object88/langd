package main

import (
	"fmt"
	"os"

	"github.com/object88/langd/examples/echo/repeater"
)

func main() {
	s := os.Args[1]
	s2 := repeater.Repeat(s)
	fmt.Printf("%s\n", s2)
}
