package tui

import (
	"fmt"
	"strings"
)

// confirmDecision is the outcome of a key press at the kill/term/pause/resume
// confirmation prompt.
type confirmDecision int

const (
	confirmIgnore  confirmDecision = iota // key not recognized; prompt stays open
	confirmExecute                        // user confirmed (y)
	confirmCancel                         // user declined (n / esc)
)

// actionMenuSelect maps a key pressed while the action menu is open to the
// resulting pending action and whether the menu should close. close==true with
// pending==actionNone means the menu was cancelled; close==false means the key
// was ignored and the menu stays open.
func actionMenuSelect(key string) (pending actionKind, closeMenu bool) {
	switch key {
	case "k", "K":
		return actionKill, true
	case "t", "T":
		return actionTerm, true
	case "p", "P":
		return actionPause, true
	case "r", "R":
		return actionResume, true
	case "n", "N":
		return actionRenice, true
	case "esc", "q", "Q":
		return actionNone, true
	default:
		return actionNone, false
	}
}

// confirmKey maps a key pressed at the confirmation prompt to a decision.
func confirmKey(key string) confirmDecision {
	switch key {
	case "y", "Y":
		return confirmExecute
	case "n", "N", "esc":
		return confirmCancel
	default:
		return confirmIgnore
	}
}

// validateNiceValue parses s as a renice value and verifies it is within the
// allowed −20…19 range. Validating here, before the syscall, lets the TUI
// reject bad input with a clear message.
func validateNiceValue(s string) (int, error) {
	var v int
	if _, err := fmt.Sscanf(strings.TrimSpace(s), "%d", &v); err != nil {
		return 0, fmt.Errorf("not a number")
	}
	if v < -20 || v > 19 {
		return 0, fmt.Errorf("out of range (−20…19)")
	}
	return v, nil
}
