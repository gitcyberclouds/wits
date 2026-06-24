package output

import (
	"fmt"
	"io"
)

type ansiString string

// Printer writes terminal-safe output to an io.Writer, sanitizing any
// string-like arguments (string, []byte, error, fmt.Stringer). Sanitization
// escapes embedded newlines/tabs so an untrusted value can't forge extra
// layout lines; the tool's own layout newlines must therefore go through
// Printf's format string (which is not sanitized), never a Print/Println arg.
type Printer struct {
	w io.Writer
}

func NewPrinter(w io.Writer) Printer {
	return Printer{w: w}
}

func (p Printer) Printf(format string, args ...any) {
	fmt.Fprintf(p.w, format, sanitizePrintArgs(args)...)
}

func (p Printer) Print(args ...any) {
	fmt.Fprint(p.w, sanitizePrintArgs(args)...)
}

func (p Printer) Println(args ...any) {
	fmt.Fprintln(p.w, sanitizePrintArgs(args)...)
}

func sanitizePrintArgs(args []any) []any {
	if len(args) == 0 {
		return nil
	}
	out := make([]any, len(args))
	for i, a := range args {
		switch v := a.(type) {
		case ansiString: // our own ansiString type is allowed to render as-is
			out[i] = string(v)
		case string:
			out[i] = SanitizeTerminalLine(v)
		case []byte:
			out[i] = SanitizeTerminalLine(string(v))
		case error:
			out[i] = SanitizeTerminalLine(v.Error())
		case fmt.Stringer:
			out[i] = SanitizeTerminalLine(v.String())
		default:
			out[i] = a
		}
	}
	return out
}
