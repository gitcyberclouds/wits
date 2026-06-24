package output

import (
	"bytes"
	"strings"
	"testing"
)

func TestSafeTerminalWriter(t *testing.T) {
	var buf bytes.Buffer
	w := NewSafeTerminalWriter(&buf)

	// An empty write is a no-op.
	if n, err := w.Write(nil); n != 0 || err != nil {
		t.Errorf("empty write = (%d, %v), want (0, nil)", n, err)
	}

	// Control characters are sanitized, but Write reports the original length so
	// callers (e.g. io.Copy) see all their bytes consumed.
	in := []byte("hi\x07there")
	n, err := w.Write(in)
	if err != nil {
		t.Fatalf("Write: %v", err)
	}
	if n != len(in) {
		t.Errorf("Write returned n=%d, want %d (the original byte count)", n, len(in))
	}
	out := buf.String()
	if strings.ContainsRune(out, '\x07') {
		t.Errorf("BEL should have been sanitized; got %q", out)
	}
	if !strings.Contains(out, "hi") || !strings.Contains(out, "there") {
		t.Errorf("surrounding text should survive sanitizing; got %q", out)
	}
}
