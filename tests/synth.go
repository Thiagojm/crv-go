//go:build ignore

package main

import (
	"fmt"
	"os"

	"github.com/Thiagojm/crv-go/internal/catalog"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, "usage: go run tests/synth.go <dir> [n]")
		os.Exit(2)
	}
	n := 4
	if len(os.Args) >= 3 {
		fmt.Sscanf(os.Args[2], "%d", &n)
	}
	if _, err := catalog.WriteSynthetic(os.Args[1], n); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
