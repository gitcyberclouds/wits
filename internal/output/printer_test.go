package output

import (
	"bytes"
	"errors"
	"strings"
	"testing"
)

type stubStringer struct{ s string }

func (s stubStringer) String() string { return s.s }

// TestPrinterSanitizesArgTypes covers the per-type sanitization branches: the
// printer must scrub control characters from string, []byte, error, and
// fmt.Stringer arguments alike.
func TestPrinterSanitizesArgTypes(t *testing.T) {
	var buf bytes.Buffer
	p := NewPrinter(&buf)

	p.Printf("a=%s b=%s c=%s d=%s\n",
		[]byte("by\x07te"),
		errors.New("er\x07ror"),
		stubStringer{"str\x07ing"},
		"pl\x07ain",
	)

	out := buf.String()
	if strings.ContainsRune(out, '\x07') {
		t.Errorf("a raw control character survived sanitization: %q", out)
	}
	// Each of the four argument types should have had its BEL escaped to a
	// literal "\x07", so the escape appears once per argument.
	if got := strings.Count(out, `\x07`); got != 4 {
		t.Errorf("expected 4 escaped control chars (one per arg type), got %d: %q", got, out)
	}
}

// TestPrinterEscapesNewlineInArg ensures a string argument can't smuggle a line
// break into the output: the only newline emitted must be the one from the
// constant format string, so an attacker-controlled field cannot forge a row.
func TestPrinterEscapesNewlineInArg(t *testing.T) {
	var buf bytes.Buffer
	p := NewPrinter(&buf)

	p.Printf("Command     : %s\n", "real\nWarnings    : none")

	out := buf.String()
	if strings.Count(out, "\n") != 1 {
		t.Errorf("arg newline was not escaped, output spans extra lines: %q", out)
	}
	if !strings.Contains(out, `real\nWarnings`) {
		t.Errorf("expected the embedded newline to render as a literal \\n: %q", out)
	}
}
