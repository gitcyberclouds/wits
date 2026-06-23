package tui

import "testing"

func TestStripAnsi(t *testing.T) {
	// Text without escape sequences passes through untouched.
	for _, s := range []string{"plain text", "PID 1234", ""} {
		if got := stripAnsi(s); got != s {
			t.Errorf("stripAnsi(%q) = %q, want unchanged", s, got)
		}
	}

	// Standard SGR/CSI color sequences are stripped, leaving only the text.
	tests := []struct {
		in   string
		want string
	}{
		{"\x1b[94mX\x1b[0m", "X"},                 // 16-color foreground
		{"\x1b[38;2;1;2;3mY\x1b[0m", "Y"},         // 24-bit truecolor foreground
		{"\x1b[1;31mPID 1234\x1b[0m", "PID 1234"}, // bold + color around a row
		{"\x1b[0m", ""},                           // bare reset
		{"a\x1b[32mb\x1b[0mc", "abc"},             // color spliced mid-string
	}
	for _, tt := range tests {
		if got := stripAnsi(tt.in); got != tt.want {
			t.Errorf("stripAnsi(%q) = %q, want %q", tt.in, got, tt.want)
		}
	}
}

func TestNormalizeRow(t *testing.T) {
	tests := []struct {
		in   string
		want string
	}{
		{"  a   b  c ", "a b c"},
		{"single", "single"},
		{"\t tabbed \t row ", "tabbed row"},
		{"   ", ""},
		{"", ""},
	}
	for _, tt := range tests {
		if got := normalizeRow(tt.in); got != tt.want {
			t.Errorf("normalizeRow(%q) = %q, want %q", tt.in, got, tt.want)
		}
	}
}
