//go:build linux || darwin || freebsd || windows

package target

import (
	"net"
	"os"
	"os/exec"
	"runtime"
	"slices"
	"testing"
	"time"
)

// Integration smoke tests for the target package's platform-specific
// resolution paths. They spin up real OS state (a sleeper child, a TCP
// listener) the test can identify by PID, then confirm the resolver finds it.

// startSleeper spawns a long-running child process suitable for name-resolution
// assertions. Returns the child's PID and process name; the caller must invoke
// the returned cleanup func to kill it.
func startSleeper(t *testing.T) (pid int, name string, cleanup func()) {
	t.Helper()

	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "windows":
		// `ping -n N 127.0.0.1` is a reliable cross-version Windows sleeper
		// that doesn't require a TTY (unlike `timeout /nobreak`).
		cmd = exec.Command("ping", "-n", "60", "127.0.0.1")
		name = "ping"
	default:
		cmd = exec.Command("sleep", "60")
		name = "sleep"
	}

	if err := cmd.Start(); err != nil {
		t.Skipf("could not spawn %s for integration test: %v", name, err)
	}
	return cmd.Process.Pid, name, func() {
		_ = cmd.Process.Kill()
		_, _ = cmd.Process.Wait()
	}
}

// TestIntegration_ResolveNameFindsSpawnedChild proves the name resolver talks
// to the real OS process table. We can't ResolveName against the test binary
// itself because every platform's name resolver excludes self+ancestors —
// that's a feature (so `witr bash` doesn't always match the user's own
// shell), but it forces this test to use a spawned child instead.
func TestIntegration_ResolveNameFindsSpawnedChild(t *testing.T) {
	childPID, name, cleanup := startSleeper(t)
	defer cleanup()

	// Give the OS a beat to register the child. ToolHelp32 / /proc updates
	// are usually instant, but CI runners can be sluggish.
	var pids []int
	var lastErr error
	for i := 0; i < 10; i++ {
		var err error
		pids, err = ResolveName(name, true)
		if err == nil && slices.Contains(pids, childPID) {
			return
		}
		lastErr = err
		time.Sleep(100 * time.Millisecond)
	}
	t.Errorf("ResolveName(%q) did not find spawned child PID %d after retries; got %v (last err: %v)",
		name, childPID, pids, lastErr)
}

// TestIntegration_ResolveNameFindsGrep is a regression test for the removed
// "grep" name-matching exclusion. witr reads the process table directly (no
// `ps | grep` pipeline), so a real grep process must be resolvable by name —
// previously it was silently filtered out on every Unix platform.
func TestIntegration_ResolveNameFindsGrep(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("grep is not available on Windows")
	}
	if _, err := exec.LookPath("grep"); err != nil {
		t.Skipf("grep not found in PATH: %v", err)
	}

	// grep with no file argument blocks reading stdin; keeping the pipe open
	// holds it alive for the duration of the lookup.
	cmd := exec.Command("grep", "witr-regression-pattern")
	stdin, err := cmd.StdinPipe()
	if err != nil {
		t.Fatalf("StdinPipe: %v", err)
	}
	if err := cmd.Start(); err != nil {
		t.Skipf("could not spawn grep: %v", err)
	}
	childPID := cmd.Process.Pid
	defer func() {
		_ = stdin.Close()
		_ = cmd.Process.Kill()
		_, _ = cmd.Process.Wait()
	}()

	var pids []int
	var lastErr error
	for i := 0; i < 10; i++ {
		pids, lastErr = ResolveName("grep", true)
		if lastErr == nil && slices.Contains(pids, childPID) {
			return
		}
		time.Sleep(100 * time.Millisecond)
	}
	t.Errorf("ResolveName(%q) did not find spawned grep PID %d after retries; got %v (last err: %v)",
		"grep", childPID, pids, lastErr)
}

// TestIntegration_ResolvePortFindsLoopbackListener binds a real TCP listener
// on a random localhost port and asserts ResolvePort attributes it to the
// test process. This drives the platform's port-resolution machinery end to
// end (netstat/lsof/sockstat/Win32 netstat plus /proc fd scan).
func TestIntegration_ResolvePortFindsLoopbackListener(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("net.Listen: %v", err)
	}
	defer ln.Close()

	port := ln.Addr().(*net.TCPAddr).Port
	self := os.Getpid()

	pids, err := ResolvePort(port)
	if err != nil {
		t.Fatalf("ResolvePort(%d): %v", port, err)
	}
	if !slices.Contains(pids, self) {
		t.Errorf("ResolvePort(%d) = %v, missing self PID %d", port, pids, self)
	}
}

// TestIntegration_ResolvePortNonexistent asserts the resolver errors cleanly
// on a port nothing has bound. Picking a random ephemeral port and closing
// the listener gives us a port we know nobody else owns.
func TestIntegration_ResolvePortNonexistent(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("net.Listen: %v", err)
	}
	port := ln.Addr().(*net.TCPAddr).Port
	ln.Close() // free the port immediately

	// The kernel may keep the port in TIME_WAIT briefly. Loop a few times
	// until the port is genuinely unowned.
	for i := 0; i < 10; i++ {
		_, err := ResolvePort(port)
		if err != nil {
			return // expected
		}
		time.Sleep(50 * time.Millisecond)
	}
	t.Errorf("ResolvePort(%d) returned nil error for an unbound port", port)
}

// TestIntegration_ResolveFileSelf creates a file, holds it open, and confirms
// the platform's file resolver attributes it to the test process — exercising
// /proc/fd (Linux), lsof (macOS), fstat (FreeBSD), and the Restart Manager
// (Windows). macOS/FreeBSD lean on external tools that may be absent or quirky
// in minimal CI images, so a miss there is tolerated rather than failed.
func TestIntegration_ResolveFileSelf(t *testing.T) {
	f, err := os.CreateTemp("", "witr-file-*.tmp")
	if err != nil {
		t.Fatalf("CreateTemp: %v", err)
	}
	defer func() { _ = os.Remove(f.Name()) }()
	defer f.Close()
	if _, err := f.WriteString("witr"); err != nil {
		t.Fatalf("write temp file: %v", err)
	}

	self := os.Getpid()
	var pids []int
	var lastErr error
	for i := 0; i < 10; i++ {
		pids, lastErr = ResolveFile(f.Name())
		if lastErr == nil && slices.Contains(pids, self) {
			return
		}
		time.Sleep(100 * time.Millisecond)
	}
	if runtime.GOOS == "darwin" || runtime.GOOS == "freebsd" {
		t.Skipf("ResolveFile on %s did not attribute the file to self (got %v, err %v); tolerated", runtime.GOOS, pids, lastErr)
	}
	t.Errorf("ResolveFile(%q) did not find self PID %d; got %v (last err: %v)", f.Name(), self, pids, lastErr)
}
