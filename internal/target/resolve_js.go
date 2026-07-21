//go:build js

// resolve_js.go resolves name/port/file targets against the in-memory world
// (see internal/proc/world_js.go) so the WASM build routes through witr's real
// resolve → analyze → render path.
package target

import (
	"fmt"
	"sort"
	"strings"

	procpkg "github.com/pranshuparmar/witr/internal/proc"
)

func ResolveName(name string, exact bool) ([]int, error) {
	procs, _ := procpkg.ListProcessSnapshot()
	lower := strings.ToLower(name)
	var pids []int
	for _, p := range procs {
		comm := strings.ToLower(p.Command)
		cmd := strings.ToLower(p.Cmdline)
		var match bool
		if exact {
			match = comm == lower
		} else {
			match = strings.Contains(comm, lower) || strings.Contains(cmd, lower)
		}
		if match {
			pids = append(pids, p.PID)
		}
	}
	sort.Ints(pids)
	if len(pids) == 0 {
		return nil, fmt.Errorf("no running process or service named %q", name)
	}
	return pids, nil
}

func ResolvePort(port int) ([]int, error) {
	ports, _ := procpkg.ListOpenPorts()
	seen := map[int]bool{}
	var pids []int
	found := false
	for _, op := range ports {
		if op.Port != port {
			continue
		}
		found = true
		if op.PID > 0 && !seen[op.PID] {
			seen[op.PID] = true
			pids = append(pids, op.PID)
		}
	}
	sort.Ints(pids)
	if len(pids) == 0 {
		if found {
			// Port is bound but no host-visible owner — triggers the container
			// fallback, exactly like a container-published port on a real host.
			return nil, ErrSocketOwnerUnknown
		}
		return nil, fmt.Errorf("no process listening on port %d", port)
	}
	return pids, nil
}

func ResolveFile(path string) ([]int, error) {
	seen := map[int]bool{}
	var pids []int
	for _, l := range procpkg.ListLockedFiles() {
		if l == nil || l.Path != path {
			continue
		}
		if !seen[l.PID] {
			seen[l.PID] = true
			pids = append(pids, l.PID)
		}
	}
	sort.Ints(pids)
	if len(pids) == 0 {
		return nil, fmt.Errorf("no process holds %s open", path)
	}
	return pids, nil
}
