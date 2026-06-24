package proc

import (
	"testing"
)

func TestDeriveDisplayCommand(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		comm    string
		cmdline string
		want    string
	}{
		{
			name:    "falls back to executable when ps truncates name",
			comm:    "AccessibilityVis",
			cmdline: "/System/Library/PrivateFrameworks/AccessibilitySupport.framework/Versions/A/Resources/AccessibilityVisualsAgent.app/Contents/MacOS/AccessibilityVisualsAgent",
			want:    "AccessibilityVisualsAgent",
		},
		{
			name:    "keeps comm when executable does not share prefix",
			comm:    "python3",
			cmdline: "python3 /tmp/script.py",
			want:    "python3",
		},
		{
			name:    "uses executable when comm empty",
			comm:    "",
			cmdline: "\"/Applications/App Name/MyBinary\" --flag",
			want:    "MyBinary",
		},
		{
			name:    "ignores env assignments before executable",
			comm:    "AccessibilityUIServer",
			cmdline: "PATH=/usr/bin /System/Library/CoreServices/AccessibilityUIServer.app/Contents/MacOS/AccessibilityUIServer",
			want:    "AccessibilityUIServer",
		},
		{
			name:    "recovers truncated Linux comm (15 char limit)",
			comm:    "my-very-long-pr",
			cmdline: "/usr/local/bin/my-very-long-process-name --daemon",
			want:    "my-very-long-process-name",
		},
		{
			name:    "handles nginx-style cmdline where first token has colon",
			comm:    "nginx",
			cmdline: "nginx: master process /usr/sbin/nginx",
			want:    "nginx:",
		},
		{
			name:    "keeps comm when it matches exe exactly",
			comm:    "nginx",
			cmdline: "/usr/sbin/nginx -g daemon off;",
			want:    "nginx",
		},
		{
			name:    "returns comm when both empty",
			comm:    "",
			cmdline: "",
			want:    "",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := deriveDisplayCommand(tt.comm, tt.cmdline); got != tt.want {
				t.Fatalf("deriveDisplayCommand(%q, %q) = %q, want %q", tt.comm, tt.cmdline, got, tt.want)
			}
		})
	}
}

func TestContainsWholeWord(t *testing.T) {
	t.Parallel()

	tests := []struct {
		s, word string
		want    bool
	}{
		{"pid 12 sleep", "12", true},
		{"pid 120 sleep", "12", false},
		{"pid 312 sleep", "12", false},
		{"12 sleep", "12", true},
		{"sleep 12", "12", true},
		{"(12)", "12", true},
		{"pid:12:sleep", "12", true},
		{"", "12", false},
		{"no match here", "12", false},
	}

	for _, tt := range tests {
		if got := containsWholeWord(tt.s, tt.word); got != tt.want {
			t.Errorf("containsWholeWord(%q, %q) = %v, want %v", tt.s, tt.word, got, tt.want)
		}
	}
}

func TestExtractExecutableName(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		cmdline string
		want    string
	}{
		{
			name:    "handles quoted path with spaces",
			cmdline: "\"/Applications/Visual Tool.app/Contents/MacOS/Visual Tool\" --flag",
			want:    "Visual Tool",
		},
		{
			name:    "skips env assignment tokens",
			cmdline: "FOO=bar BAR=baz /usr/local/bin/server --mode production",
			want:    "server",
		},
		{
			name:    "returns empty when no executable found",
			cmdline: "",
			want:    "",
		},
		{
			name:    "handles simple command",
			cmdline: "/usr/bin/my-very-long-process-name --flag",
			want:    "my-very-long-process-name",
		},
		{
			// Documents the known limitation that caused issue #201: when `ps`
			// emits an unquoted argv (which it always does on darwin/freebsd),
			// a path containing spaces is tokenized incorrectly. Callers must
			// use binaryBasename(comm) or binaryBasename(binPath) to get the
			// correct display name in this case — extractExecutableName cannot
			// recover it from raw args alone.
			name:    "loses spaces in unquoted .app path (issue #201)",
			cmdline: "/Applications/Microsoft Teams.app/Contents/MacOS/Microsoft Teams --arg",
			want:    "Microsoft",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := extractExecutableName(tt.cmdline); got != tt.want {
				t.Fatalf("extractExecutableName(%q) = %q, want %q", tt.cmdline, got, tt.want)
			}
		})
	}
}

func TestBinaryBasename(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		in   string
		want string
	}{
		{
			name: "preserves spaces in .app path",
			in:   "/Applications/Microsoft Teams.app/Contents/MacOS/Microsoft Teams",
			want: "Microsoft Teams",
		},
		{
			name: "preserves spaces in helper renderer path",
			in:   "/Applications/Google Chrome.app/Contents/Frameworks/Google Chrome Framework.framework/Versions/Current/Helpers/Google Chrome Helper (Renderer).app/Contents/MacOS/Google Chrome Helper (Renderer)",
			want: "Google Chrome Helper (Renderer)",
		},
		{
			name: "trims surrounding whitespace and newline (ps -o comm= output)",
			in:   "/Applications/Visual Studio Code.app/Contents/MacOS/Electron\n",
			want: "Electron",
		},
		{
			name: "strips surrounding quotes",
			in:   `"/usr/local/bin/my server"`,
			want: "my server",
		},
		{
			name: "returns empty for empty input",
			in:   "",
			want: "",
		},
		{
			name: "returns empty for whitespace-only input",
			in:   "   \n  ",
			want: "",
		},
		{
			name: "rejects bare slash",
			in:   "/",
			want: "",
		},
		{
			name: "handles simple basename without path",
			in:   "bash",
			want: "bash",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := binaryBasename(tt.in); got != tt.want {
				t.Fatalf("binaryBasename(%q) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}
