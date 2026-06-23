//go:build linux

package proc

import (
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"
	"testing"
)

// These tests anchor on the test process itself (its PID, an fd it holds, a
// lock it takes, a port it binds) so the assertions are deterministic even
// though the real /proc has hundreds of unrelated entries.

func TestReadExtendedInfoSelf(t *testing.T) {
	mem, _, fds, fdCount, fdLimit, threads, err := ReadExtendedInfo(os.Getpid())
	if err != nil {
		t.Fatalf("ReadExtendedInfo(self): %v", err)
	}
	if mem.RSS == 0 {
		t.Error("RSS should be non-zero for a running process")
	}
	if fdCount == 0 || len(fds) == 0 {
		t.Errorf("fdCount=%d, len(fds)=%d; both should be > 0 (stdin/out/err at minimum)", fdCount, len(fds))
	}
	if fdLimit == 0 {
		t.Error("fdLimit should be > 0")
	}
	if threads == 0 {
		t.Error("threadCount should be > 0 (the Go runtime is multi-threaded)")
	}
}

func TestGetFileContextSelf(t *testing.T) {
	fc := GetFileContext(os.Getpid())
	if fc == nil {
		t.Fatal("GetFileContext(self) = nil")
	}
	if fc.OpenFiles == 0 {
		t.Error("OpenFiles should be > 0")
	}
	if fc.FileLimit == 0 {
		t.Error("FileLimit should be > 0")
	}
}

func TestGetResourceContextSelf(t *testing.T) {
	// We can't assert specific resource numbers (they depend on the host), but
	// the call must always return a non-nil context and exercise every reader.
	if GetResourceContext(os.Getpid()) == nil {
		t.Error("GetResourceContext(self) should never be nil")
	}
}

func TestGetCmdlineSelf(t *testing.T) {
	if cmd := GetCmdline(os.Getpid()); cmd == "" || cmd == "(unknown)" {
		t.Errorf("GetCmdline(self) = %q, want the test binary's command line", cmd)
	}
	if cmd := GetCmdline(0); cmd != "(unknown)" {
		t.Errorf("GetCmdline(0) = %q, want \"(unknown)\" for the unreadable kernel PID", cmd)
	}
}

func TestListProcessSnapshotIncludesSelf(t *testing.T) {
	snap, err := ListProcessSnapshot()
	if err != nil {
		t.Fatalf("ListProcessSnapshot: %v", err)
	}
	self := os.Getpid()
	for _, p := range snap {
		if p.PID == self {
			return
		}
	}
	t.Errorf("snapshot of %d processes did not include self PID %d", len(snap), self)
}

func TestResolveChildrenFindsSpawnedChild(t *testing.T) {
	if _, err := ResolveChildren(0); err == nil {
		t.Error("ResolveChildren(0) should reject the invalid PID")
	}

	c := exec.Command("sleep", "30")
	if err := c.Start(); err != nil {
		t.Skipf("cannot spawn child: %v", err)
	}
	defer func() {
		_ = c.Process.Kill()
		_ = c.Wait()
	}()

	kids, err := ResolveChildren(os.Getpid())
	if err != nil {
		t.Fatalf("ResolveChildren(self): %v", err)
	}
	for _, k := range kids {
		if k.PID == c.Process.Pid {
			return
		}
	}
	t.Errorf("ResolveChildren(self) did not include the spawned child %d (got %d children)", c.Process.Pid, len(kids))
}

func TestListAllOpenFilesFindsHeldFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "witr-open.txt")
	f, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	self := os.Getpid()
	for _, of := range ListAllOpenFiles() {
		if of.PID == self && of.Path == path {
			if of.Type != "OPEN" {
				t.Errorf("Type = %q, want OPEN", of.Type)
			}
			return
		}
	}
	t.Errorf("ListAllOpenFiles did not include our open file %q owned by PID %d", path, self)
}

func TestListLockedFilesFindsHeldLock(t *testing.T) {
	f, err := os.CreateTemp(t.TempDir(), "witr-lock")
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	// Take a real POSIX write lock so it shows up in /proc/locks under our PID.
	lk := syscall.Flock_t{Type: syscall.F_WRLCK, Whence: 0}
	if err := syscall.FcntlFlock(f.Fd(), syscall.F_SETLK, &lk); err != nil {
		t.Skipf("cannot place POSIX lock: %v", err)
	}

	self := os.Getpid()
	for _, l := range ListLockedFiles() {
		if l.PID == self {
			return // our lock surfaced — resolveLockPath/lockProcessName/statKey exercised
		}
	}
	t.Errorf("ListLockedFiles did not include a lock held by self (PID %d)", self)
}

func TestGetSocketStateForPortFindsListener(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("net.Listen: %v", err)
	}
	defer ln.Close()

	port := ln.Addr().(*net.TCPAddr).Port
	info := GetSocketStateForPort(port)
	if info == nil {
		t.Fatalf("GetSocketStateForPort(%d) = nil, want our listener", port)
	}
	if info.State != "LISTEN" {
		t.Errorf("socket state = %q, want LISTEN", info.State)
	}
}
