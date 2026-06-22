package app

import (
	"errors"
	"testing"
)

// The mappings here drive script and CI integrations, so any
// regression in classification is a breaking change for users.
func TestClassifyError(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		msg  string
		want int
	}{
		{"permission denied → 3", "open /proc/1/environ: permission denied", ExitPermission},
		{"operation not permitted → 3", "kill: operation not permitted", ExitPermission},
		{"insufficient permissions → 3", "insufficient permissions to read process info", ExitPermission},

		{"no matching → 2", "no matching process found", ExitNotFound},
		{"no running process → 2", "no running process with that name", ExitNotFound},
		{"not found → 2", "process 999999 not found", ExitNotFound},
		{"no process → 2", "no process listening on port 80", ExitNotFound},

		{"invalid → 4", "invalid pid", ExitInvalidInput},
		{"must specify → 4", "must specify --pid, --port, --file, or a process name", ExitInvalidInput},

		{"unknown error → internal", "something exploded", ExitInternalError},
		{"empty message → internal", "", ExitInternalError},

		{"case-insensitive (uppercase)", "PERMISSION DENIED", ExitPermission},
		{"case-insensitive (mixed)", "No Matching Process", ExitNotFound},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := classifyError(errors.New(tt.msg))
			if got != tt.want {
				t.Errorf("classifyError(%q) = %d, want %d", tt.msg, got, tt.want)
			}
		})
	}
}

// An internal error must be distinguishable from "process has warnings" so
// scripts gating on exit 1 don't conflate the two.
func TestExitCodesDistinct(t *testing.T) {
	t.Parallel()
	if ExitInternalError == ExitWarnings {
		t.Errorf("ExitInternalError (%d) must differ from ExitWarnings (%d)", ExitInternalError, ExitWarnings)
	}
	if ExitInternalError != 5 {
		t.Errorf("ExitInternalError = %d, want 5 (documented)", ExitInternalError)
	}
}
