//go:build windows

package proc

import (
	"os"
	"path/filepath"
	"testing"
)

// TestIsWindowsBinaryDeleted guards the issue #205 false positive: protected
// processes (e.g. vmmemWSL) only expose a bare image name via the process
// snapshot, which must be treated as "path unknown", not "binary deleted".
func TestIsWindowsBinaryDeleted(t *testing.T) {
	// Bare image names and empty strings are not confirmed-deleted binaries.
	for _, name := range []string{"", "vmmemWSL", "System", "Registry", "wslservice.exe"} {
		if isWindowsBinaryDeleted(name) {
			t.Errorf("isWindowsBinaryDeleted(%q) = true; bare/empty names must be treated as unknown, not deleted", name)
		}
	}

	// The running test binary is a real, existing absolute path.
	exe, err := os.Executable()
	if err != nil {
		t.Fatalf("os.Executable: %v", err)
	}
	if isWindowsBinaryDeleted(exe) {
		t.Errorf("isWindowsBinaryDeleted(%q) = true; the running test binary exists", exe)
	}

	// An absolute path that does not exist is a genuine deleted binary.
	missing := filepath.Join(os.TempDir(), "witr-nonexistent-binary-xyz.exe")
	if !isWindowsBinaryDeleted(missing) {
		t.Errorf("isWindowsBinaryDeleted(%q) = false; want true for a missing absolute path", missing)
	}
}
