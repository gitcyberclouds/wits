//go:build darwin

package proc

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

func TestGetCwdAndBinaryPath(t *testing.T) {
	tmpDir := t.TempDir()
	binPath := filepath.Join(tmpDir, "fakebin")
	cwdPath := filepath.Join(tmpDir, "fakecwd")
	if err := os.WriteFile(binPath, []byte("ok"), 0o644); err != nil {
		t.Fatalf("write file: %v", err)
	}
	if err := os.Mkdir(cwdPath, 0o755); err != nil {
		t.Fatalf("mkdir cwd: %v", err)
	}

	// Create a fake lsof command that emits both cwd and txt entries.
	fakeBinDir := filepath.Join(tmpDir, "bin")
	if err := os.Mkdir(fakeBinDir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	lsofScript := filepath.Join(fakeBinDir, "lsof")
	script := fmt.Sprintf("#!/bin/sh\nprintf 'p123\\nfcwd\\nn%s\\nftxt\\nn%s\\n'", cwdPath, binPath)
	if err := os.WriteFile(lsofScript, []byte(script), 0o755); err != nil {
		t.Fatalf("write lsof script: %v", err)
	}

	t.Setenv("PATH", fakeBinDir+":"+os.Getenv("PATH"))

	cwd, bin := getCwdAndBinaryPath(123)
	if cwd != cwdPath {
		t.Fatalf("getCwdAndBinaryPath() cwd = %q, want %q", cwd, cwdPath)
	}
	if bin != binPath {
		t.Fatalf("getCwdAndBinaryPath() binPath = %q, want %q", bin, binPath)
	}

	// Binary exists — not deleted
	_, err := os.Stat(bin)
	if os.IsNotExist(err) {
		t.Fatalf("expected binary to exist")
	}

	// Delete binary, verify stat detects it
	if err := os.Remove(binPath); err != nil {
		t.Fatalf("rm: %v", err)
	}
	_, err = os.Stat(bin)
	if !os.IsNotExist(err) {
		t.Fatalf("expected binary to be detected as deleted")
	}
}
