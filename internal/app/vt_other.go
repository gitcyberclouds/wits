//go:build !windows

package app

import "io"

// enableVirtualTerminal is a no-op: Unix terminals interpret ANSI escape
// sequences natively.
func enableVirtualTerminal(w io.Writer) {}
