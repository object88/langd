package main

import (
	"os"

	"github.com/object88/langd/examples/echo/repeater"
)

func main() {
	s := os.Args[1]
	repeater.Repeat(s)
}
