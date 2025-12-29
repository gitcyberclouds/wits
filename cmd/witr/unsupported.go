//go:build !linux && !darwin

package main

import (
	"fmt"
	"os"
)

func main() {
	fmt.Fprintln(
		os.Stderr,
		"witr is only supported on Linux and macOS.\n\nIf you are seeing this message, you are attempting to build or run witr on an unsupported platform (such as Windows).\n\nPlease use Linux or macOS to build and run witr.",
	)
	os.Exit(1)
}
