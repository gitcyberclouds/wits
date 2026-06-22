package app

import (
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// buildWitr compiles the real witr binary once so the process exit codes can be
// characterized end to end (Execute -> runApp -> os.Exit). The exit-code
// contract drives scripting and CI integrations, so it's asserted against the
// actual binary rather than internal helpers.
func buildWitr(t *testing.T) string {
	t.Helper()

	gomod, err := exec.Command("go", "env", "GOMOD").Output()
	if err != nil {
		t.Fatalf("go env GOMOD: %v", err)
	}
	root := filepath.Dir(strings.TrimSpace(string(gomod)))

	bin := filepath.Join(t.TempDir(), "witr")
	if runtime.GOOS == "windows" {
		bin += ".exe"
	}
	build := exec.Command("go", "build", "-o", bin, "./cmd/witr")
	build.Dir = root
	if out, err := build.CombinedOutput(); err != nil {
		t.Fatalf("build witr: %v\n%s", err, out)
	}
	return bin
}

func runExit(t *testing.T, bin string, args ...string) int {
	t.Helper()
	err := exec.Command(bin, args...).Run()
	if err == nil {
		return 0
	}
	if ee, ok := err.(*exec.ExitError); ok {
		return ee.ExitCode()
	}
	t.Fatalf("run witr %v: %v", args, err)
	return -1
}

func TestExitCodes(t *testing.T) {
	if testing.Short() {
		t.Skip("builds the witr binary; skipped under -short")
	}
	bin := buildWitr(t)

	// PID 2147483646 is far above any real PID on Linux/macOS/Windows, so the
	// "not found" path is deterministic on CI runners.
	const ghostPID = "2147483646"

	tests := []struct {
		name string
		args []string
		want int
	}{
		{"invalid pid (non-numeric)", []string{"--pid", "notanumber"}, ExitInvalidInput},
		{"invalid pid (zero)", []string{"--pid", "0"}, ExitInvalidInput},
		{"invalid port (out of range)", []string{"--port", "70000"}, ExitInvalidInput},
		{"not found (ghost pid)", []string{"--pid", ghostPID}, ExitNotFound},
		// Multi-target exit code is the highest severity among targets, not the
		// first or last — assert with both orderings of a not-found(2) and an
		// invalid(4) target.
		{"multi: not-found then invalid", []string{"--pid", ghostPID, "--port", "70000"}, ExitInvalidInput},
		{"multi: invalid then not-found", []string{"--port", "70000", "--pid", ghostPID}, ExitInvalidInput},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			if got := runExit(t, bin, tc.args...); got != tc.want {
				t.Errorf("witr %v exit = %d, want %d", tc.args, got, tc.want)
			}
		})
	}
}
