package app

import (
	"io"
	"os"

	"github.com/mattn/go-isatty"
)

// useColor reports whether CLI output should be colorized for the given writer.
// Color is enabled only when the user hasn't disabled it (--no-color or the
// NO_COLOR convention) AND the destination is an interactive terminal — so
// piping or redirecting to a file yields clean text with no escape codes. When
// color is enabled it also ensures the terminal will interpret the sequences (a
// no-op outside Windows). The terminal setup is idempotent, so calling this on
// each render path is harmless.
func useColor(flags appFlags, w io.Writer) bool {
	if flags.noColor || os.Getenv("NO_COLOR") != "" || !isTerminal(w) {
		return false
	}
	enableVirtualTerminal(w)
	return true
}

// isTerminal reports whether w is an interactive terminal/console rather than a
// pipe or regular file.
func isTerminal(w io.Writer) bool {
	f, ok := w.(*os.File)
	if !ok {
		return false
	}
	return isatty.IsTerminal(f.Fd()) || isatty.IsCygwinTerminal(f.Fd())
}
