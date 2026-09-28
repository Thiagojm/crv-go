//go:build ignore

package main

import (
	"fmt"
	"os"

	"github.com/Thiagojm/crv-go/internal/catalog"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, "usage: go run tests/synth.go <dir>")
		os.Exit(2)
	}
	if _, err := catalog.WriteSynthetic(os.Args[1], 4); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
