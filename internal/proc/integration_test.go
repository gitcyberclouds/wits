//go:build linux || darwin || freebsd || windows

package proc

import (
	"net"
	"os"
	"testing"
)

// Integration smoke tests for the platform-specific OS plumbing. They assert
// invariants — "the call returns *something* sensible" — not exact output,
// because the real system has hundreds of processes the test cannot know
// about. The single thing we DO know about is our own test binary plus
// anything we explicitly spawn or bind to, so every assertion is anchored on
// one of those.

// TestIntegration_ListProcessesIncludesSelf confirms the enumerator is wired
// to a real source (ToolHelp32 on Windows, /proc on Linux, ps/sysctl on macOS
// and FreeBSD) by finding the test process in its output.
func TestIntegration_ListProcessesIncludesSelf(t *testing.T) {
	procs, err := ListProcesses()
	if err != nil {
		t.Fatalf("ListProcesses: %v", err)
	}
	if len(procs) == 0 {
		t.Fatalf("ListProcesses returned 0 processes; expected at least our own")
	}

	self := os.Getpid()
	for _, p := range procs {
		if p.PID == self {
			return
		}
	}
	t.Errorf("ListProcesses did not contain self PID %d (returned %d procs)", self, len(procs))
}

// TestIntegration_ReadProcessSelf asserts ReadProcess produces a non-degenerate
// record for the test binary: matching PID, a parent (anything we can spawn
// has a parent), and a non-empty command name.
func TestIntegration_ReadProcessSelf(t *testing.T) {
	p, err := ReadProcess(os.Getpid())
	if err != nil {
		t.Fatalf("ReadProcess(self): %v", err)
	}
	if p.PID != os.Getpid() {
		t.Errorf("ReadProcess(self).PID = %d, want %d", p.PID, os.Getpid())
	}
	if p.PPID == 0 {
		t.Errorf("ReadProcess(self).PPID = 0, want non-zero (every spawned process has a parent)")
	}
	if p.Command == "" {
		t.Errorf("ReadProcess(self).Command is empty")
	}
}

// TestIntegration_ReadProcessSelfMemory is the regression guard for issue #205:
// the standard (non-verbose) per-process path must populate resident memory.
// Our own process always has a non-zero working set, so MemoryRSS must be > 0
// on every platform.
func TestIntegration_ReadProcessSelfMemory(t *testing.T) {
	p, err := ReadProcess(os.Getpid())
	if err != nil {
		t.Fatalf("ReadProcess(self): %v", err)
	}
	if p.MemoryRSS == 0 {
		t.Errorf("ReadProcess(self).MemoryRSS = 0; want non-zero resident memory")
	}
}

// TestIntegration_ReadProcessNonexistent verifies error handling for a PID
// that cannot exist (PID 0 is reserved by the kernel and never represents a
// userland process).
func TestIntegration_ReadProcessNonexistent(t *testing.T) {
	if _, err := ReadProcess(0); err == nil {
		t.Errorf("ReadProcess(0) returned no error; want an error")
	}
}

// TestIntegration_ResolveAncestrySelf walks the parent chain from our PID
// and confirms the chain ends with us (every implementation should produce
// init/systemd/launchd/SCM ... → self).
func TestIntegration_ResolveAncestrySelf(t *testing.T) {
	chain, err := ResolveAncestry(os.Getpid())
	if err != nil {
		t.Fatalf("ResolveAncestry(self): %v", err)
	}
	if len(chain) == 0 {
		t.Fatalf("ResolveAncestry returned empty chain")
	}
	last := chain[len(chain)-1]
	if last.PID != os.Getpid() {
		t.Errorf("ancestry chain does not end in self PID; got %d, want %d", last.PID, os.Getpid())
	}
	// Sanity: the chain should have at least two entries (us + a parent).
	if len(chain) < 2 {
		t.Errorf("ancestry chain has only %d entries; expected at least our parent too", len(chain))
	}
}

// TestIntegration_ListOpenPortsFindsLoopbackListener binds a real TCP listener
// on a random localhost port, then asserts ListOpenPorts attributes that port
// to our test process. This is the highest-value integration test — every
// platform path (lsof on macOS, /proc on Linux, sockstat on FreeBSD, netstat
// on Windows) gets exercised end to end.
func TestIntegration_ListOpenPortsFindsLoopbackListener(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("net.Listen: %v", err)
	}
	defer ln.Close()

	port := ln.Addr().(*net.TCPAddr).Port
	self := os.Getpid()

	ports, err := ListOpenPorts()
	if err != nil {
		t.Fatalf("ListOpenPorts: %v", err)
	}

	for _, p := range ports {
		if p.Port == port && p.PID == self {
			return
		}
	}
	t.Errorf("ListOpenPorts did not find loopback listener on port %d owned by PID %d", port, self)
}
