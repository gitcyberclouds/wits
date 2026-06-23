//go:build !windows

package tui

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"testing"
	"time"
)

func TestSetNiceRangeGuard(t *testing.T) {
	// Out-of-range values are rejected before any syscall is attempted.
	if err := setNice(os.Getpid(), 100); err == nil {
		t.Error("setNice(+100) should be rejected as out of range")
	}
	if err := setNice(os.Getpid(), -100); err == nil {
		t.Error("setNice(-100) should be rejected as out of range")
	}
}

// startSleeper spawns a long-lived child to receive signals.
func startSleeper(t *testing.T) *exec.Cmd {
	t.Helper()
	c := exec.Command("sleep", "60")
	if err := c.Start(); err != nil {
		t.Skipf("cannot spawn child process: %v", err)
	}
	return c
}

// waitExit reports whether the child reaps within the timeout (i.e. the signal
// actually terminated it).
func waitExit(c *exec.Cmd, timeout time.Duration) bool {
	done := make(chan struct{})
	go func() { _ = c.Wait(); close(done) }()
	select {
	case <-done:
		return true
	case <-time.After(timeout):
		return false
	}
}

func TestSignalActionsTerminate(t *testing.T) {
	t.Run("SIGTERM exits the process", func(t *testing.T) {
		c := startSleeper(t)
		if err := termProcess(c.Process.Pid); err != nil {
			t.Fatalf("termProcess: %v", err)
		}
		if !waitExit(c, 5*time.Second) {
			_ = c.Process.Kill()
			t.Fatal("process did not exit after SIGTERM")
		}
	})

	t.Run("SIGKILL exits the process", func(t *testing.T) {
		c := startSleeper(t)
		if err := killProcess(c.Process.Pid); err != nil {
			t.Fatalf("killProcess: %v", err)
		}
		if !waitExit(c, 5*time.Second) {
			t.Fatal("process did not exit after SIGKILL")
		}
	})
}

func TestSignalActionsPauseResume(t *testing.T) {
	c := startSleeper(t)
	defer func() {
		_ = c.Process.Kill()
		_ = c.Wait()
	}()

	if err := pauseProcess(c.Process.Pid); err != nil {
		t.Fatalf("pauseProcess: %v", err)
	}
	if !waitForState(c.Process.Pid, 'T', 2*time.Second) {
		t.Error("process did not reach the stopped (T) state after SIGSTOP")
	}
	if err := resumeProcess(c.Process.Pid); err != nil {
		t.Fatalf("resumeProcess: %v", err)
	}
}

func TestSignalDeadProcessErrors(t *testing.T) {
	c := startSleeper(t)
	pid := c.Process.Pid
	_ = c.Process.Kill()
	_ = c.Wait() // reap, so the PID is no longer signalable

	if err := resumeProcess(pid); err == nil {
		t.Error("signalling a reaped PID should fail")
	}
}

// waitForState polls /proc for the given process state on Linux. On other
// platforms it returns true (the /proc check is not meaningful there), so the
// pause assertion is only enforced where it can be observed.
func waitForState(pid int, want byte, timeout time.Duration) bool {
	if runtime.GOOS != "linux" {
		return true
	}
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if data, err := os.ReadFile(fmt.Sprintf("/proc/%d/stat", pid)); err == nil {
			// Format: "pid (comm) state ..."; comm may contain spaces and
			// parens, so the state is the char two positions after the last ')'.
			if i := bytes.LastIndexByte(data, ')'); i >= 0 && i+2 < len(data) && data[i+2] == want {
				return true
			}
		}
		time.Sleep(10 * time.Millisecond)
	}
	return false
}
