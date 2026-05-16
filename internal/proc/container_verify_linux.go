//go:build linux

package proc

import (
	"fmt"
	"os"
	"strings"
)

// PIDBelongsToContainer verifies that the host process at pid actually belongs
// to the given container by checking its cgroup membership. Guards against
// the case where docker inspect returns a PID that's namespaced to a Docker
// VM (macOS, Windows) and happens to coincide with an unrelated host PID.
func PIDBelongsToContainer(pid int, containerID string) bool {
	if pid <= 0 || containerID == "" {
		return false
	}
	data, err := os.ReadFile(fmt.Sprintf("/proc/%d/cgroup", pid))
	if err != nil {
		return false
	}
	return strings.Contains(string(data), containerID)
}
