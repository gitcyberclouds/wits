//go:build linux

package proc

import (
	"fmt"
	"os"
	"strings"
)

// serviceFromCgroup derives the systemd .service unit owning pid from its cgroup
// membership (/proc/<pid>/cgroup) — a cheap file read, no subprocess. The cgroup
// file is world-readable, so this works for processes the caller does not own.
// Returns "" when the process belongs to a .scope (login session, app scope) or
// to no systemd unit.
func serviceFromCgroup(pid int) string {
	data, err := os.ReadFile(fmt.Sprintf("/proc/%d/cgroup", pid))
	if err != nil {
		return ""
	}
	return serviceUnitFromCgroup(string(data))
}

// serviceUnitFromCgroup parses /proc/<pid>/cgroup content and returns the
// deepest systemd unit the process belongs to, but only when that unit is a
// .service. A process whose nearest unit is a .scope (a session or app scope,
// or init.scope) is not a managed service, so it yields "".
func serviceUnitFromCgroup(content string) string {
	for _, line := range strings.Split(content, "\n") {
		parts := strings.SplitN(line, ":", 3)
		if len(parts) < 3 {
			continue
		}
		controllers, path := parts[1], strings.TrimSpace(parts[2])
		// cgroup v2 (unified) lines have an empty controller field; cgroup v1
		// carries a name=systemd controller. Skip other v1 hierarchies, whose
		// paths don't reflect unit membership.
		if controllers != "" && !strings.Contains(controllers, "systemd") {
			continue
		}
		segments := strings.Split(path, "/")
		for i := len(segments) - 1; i >= 0; i-- {
			switch {
			case strings.HasSuffix(segments[i], ".service"):
				return segments[i]
			case strings.HasSuffix(segments[i], ".scope"):
				return ""
			}
		}
	}
	return ""
}
