//go:build windows

package app

import (
	"io"
	"os"

	"golang.org/x/sys/windows"
)

// enableVirtualTerminal turns on ANSI escape-sequence processing for a Windows
// console so colored output renders instead of printing raw escape codes (e.g.
// "←[0m"). Some console hosts and elevated launches start with it disabled.
// It is best-effort: if the destination isn't a console, the call is skipped.
func enableVirtualTerminal(w io.Writer) {
	f, ok := w.(*os.File)
	if !ok {
		return
	}
	handle := windows.Handle(f.Fd())
	var mode uint32
	if err := windows.GetConsoleMode(handle, &mode); err != nil {
		return
	}
	_ = windows.SetConsoleMode(handle, mode|windows.ENABLE_VIRTUAL_TERMINAL_PROCESSING)
}
