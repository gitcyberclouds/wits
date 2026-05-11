package proc

import (
	"path/filepath"
	"strings"
)

// deriveDisplayCommand returns a human-readable command name that avoids
// kernel comm-field truncation (typically 15-16 chars on Linux/macOS/FreeBSD)
// by falling back to the executable name extracted from the full command line
// when the short name looks clipped.
func deriveDisplayCommand(comm, cmdline string) string {
	trimmedComm := strings.TrimSpace(comm)
	exe := extractExecutableName(cmdline)
	if trimmedComm == "" {
		return exe
	}
	if exe == "" {
		return trimmedComm
	}
	if strings.HasPrefix(exe, trimmedComm) && len(trimmedComm) < len(exe) {
		return exe
	}
	return trimmedComm
}

// containsWholeWord checks if s contains word as a standalone token,
// not as a substring of a larger number or identifier.
func containsWholeWord(s, word string) bool {
	idx := 0
	for {
		i := strings.Index(s[idx:], word)
		if i < 0 {
			return false
		}
		start := idx + i
		end := start + len(word)

		leftOK := start == 0 || !isWordChar(s[start-1])
		rightOK := end == len(s) || !isWordChar(s[end])
		if leftOK && rightOK {
			return true
		}
		idx = start + 1
	}
}

func isWordChar(c byte) bool {
	return (c >= '0' && c <= '9') || (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || c == '_'
}

func extractExecutableName(cmdline string) string {
	args := splitCmdline(cmdline)
	for _, arg := range args {
		if arg == "" {
			continue
		}
		if strings.Contains(arg, "=") && !strings.Contains(arg, "/") {
			// Skip leading environment assignments.
			continue
		}
		clean := strings.Trim(arg, "\"'")
		if clean == "" {
			continue
		}
		base := filepath.Base(clean)
		if base == "." || base == "" || base == "/" {
			continue
		}
		return base
	}
	return ""
}
