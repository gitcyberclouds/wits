//go:build !linux && !darwin && !freebsd && !windows

package main

import (
	"fmt"
	"os"
)

func main() {
	fmt.Fprintln(
		os.Stderr,
		"witr is only supported on Linux, macOS, FreeBSD, and Windows.\n\nIf you are seeing this message, you are attempting to build or run witr on an unsupported platform.\n\nPlease use Linux, macOS, FreeBSD, or Windows to build and run witr.",
	)
	os.Exit(1)
}
