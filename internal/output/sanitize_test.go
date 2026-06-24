package output

import (
	"fmt"
	"strings"
	"testing"
	"unicode"
)

func FuzzAppendEscapedRune(f *testing.F) {
	f.Add(uint32(0x00))
	f.Add(uint32(0x1b))
	f.Add(uint32(0x7f))
	f.Add(uint32(0x80))
	f.Add(uint32(0xff))
	f.Add(uint32(0x100))
	f.Add(uint32(0x20ac))
	f.Add(uint32(0xffff))
	f.Add(uint32(0x10000))
	f.Add(uint32(0x10ffff))

	f.Fuzz(func(t *testing.T, raw uint32) {
		// keep this within the valid Unicode scalar range
		r := rune(raw % (unicode.MaxRune + 1))

		var b strings.Builder
		appendEscapedRune(&b, r)
		got := b.String()

		var want string
		switch {
		case r <= 0xFF:
			want = fmt.Sprintf(`\x%02x`, r)
		case r <= 0xFFFF:
			want = fmt.Sprintf(`\u%04x`, r)
		default:
			want = fmt.Sprintf(`\U%08x`, r)
		}

		if got != want {
			t.Fatalf("appendEscapedRune(%#x) = %q, want %q", r, got, want)
		}

		// output must be visible ascii
		for i := 0; i < len(got); i++ {
			if got[i] >= 0x80 {
				t.Fatalf("appendEscapedRune(%#x) produced non-ASCII byte 0x%02x in %q", r, got[i], got)
			}
		}
	})
}

// TestSanitizeTerminalEscapesControlBytes pins the human-readable contract: a
// raw ESC renders as the visible escape \x1b (a single backslash), with the
// surrounding printable bytes left intact.
func TestSanitizeTerminalEscapesControlBytes(t *testing.T) {
	in := "a\x1b[31mb"
	want := `a\x1b[31mb`
	if got := SanitizeTerminal(in); got != want {
		t.Fatalf("SanitizeTerminal(%q) = %q, want %q", in, got, want)
	}
}

// TestSanitizeTerminalLineEscapesLineBreakers verifies the single-line variant
// neutralizes the two characters SanitizeTerminal preserves (newline and tab),
// so an attacker-controlled field can't forge extra lines or shift columns,
// while still escaping ESC like the base sanitizer.
func TestSanitizeTerminalLineEscapesLineBreakers(t *testing.T) {
	cases := map[string]string{
		"cmd\nWarnings : none": `cmd\nWarnings : none`,
		"a\tb":                 `a\tb`,
		"x\x1b[2Jy":            `x\x1b[2Jy`,
		"plain":                "plain",
	}
	for in, want := range cases {
		if got := SanitizeTerminalLine(in); got != want {
			t.Errorf("SanitizeTerminalLine(%q) = %q, want %q", in, got, want)
		}
	}
}
