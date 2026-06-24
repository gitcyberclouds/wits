//go:build linux

package proc

import "testing"

func TestIsInterestingFile(t *testing.T) {
	uninteresting := []string{
		"socket:[12345]", "pipe:[678]", "anon_inode:[eventfd]", "/memfd:foo",
		"/proc/self/status", "/sys/kernel/x", "/dev/null", "/dev/tty1", "/dev/pts/0",
	}
	for _, p := range uninteresting {
		if isInterestingFile(p) {
			t.Errorf("isInterestingFile(%q) = true, want false (kernel/internal fd)", p)
		}
	}
	interesting := []string{"/home/user/notes.txt", "/var/log/app.log", "/etc/hosts"}
	for _, p := range interesting {
		if !isInterestingFile(p) {
			t.Errorf("isInterestingFile(%q) = false, want true (real file)", p)
		}
	}
}
