//go:build linux

package proc

import (
	"context"
	"os"
	"syscall"
	"testing"
)

func TestPIDBelongsToContainer(t *testing.T) {
	if PIDBelongsToContainer(0, "abc") {
		t.Error("a non-positive PID should never belong to a container")
	}
	if PIDBelongsToContainer(os.Getpid(), "") {
		t.Error("an empty container id should never match")
	}
	// Our own cgroup won't contain a random fabricated container id.
	if PIDBelongsToContainer(os.Getpid(), "deadbeefdeadbeefdeadbeefdeadbeef") {
		t.Error("self should not belong to a fabricated container id")
	}
}

func TestGetLockedFilesFindsHeldLock(t *testing.T) {
	f, err := os.CreateTemp(t.TempDir(), "witr-flock")
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	lk := syscall.Flock_t{Type: syscall.F_WRLCK, Whence: 0}
	if err := syscall.FcntlFlock(f.Fd(), syscall.F_SETLK, &lk); err != nil {
		t.Skipf("cannot place POSIX lock: %v", err)
	}

	// getLockedFiles reads /proc/locks and resolves the held lock to the open
	// file's path via this process's own fds.
	got := getLockedFiles(os.Getpid())
	if len(got) == 0 {
		t.Fatal("getLockedFiles(self) returned nothing while holding a lock")
	}
	found := false
	for _, p := range got {
		if p == f.Name() {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("getLockedFiles(self) = %v, want it to include %q", got, f.Name())
	}
}

func TestCommandAsOriginalUser(t *testing.T) {
	// When not running under sudo (the common case), it behaves exactly like
	// exec.CommandContext: the command and args pass through unchanged.
	cmd := commandAsOriginalUser(context.Background(), "echo", "hello")
	if cmd == nil {
		t.Fatal("commandAsOriginalUser returned nil")
	}
	if len(cmd.Args) != 2 || cmd.Args[0] != "echo" || cmd.Args[1] != "hello" {
		t.Errorf("cmd.Args = %v, want [echo hello]", cmd.Args)
	}
}
