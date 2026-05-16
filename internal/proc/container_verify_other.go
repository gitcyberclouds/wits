//go:build !linux

package proc

// PIDBelongsToContainer always returns false on non-Linux platforms because
// the cgroup-based check that proves PID ownership doesn't exist there. The
// caller falls back to rendering container details directly without trusting
// the host PID.
func PIDBelongsToContainer(pid int, containerID string) bool {
	return false
}
