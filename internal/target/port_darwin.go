//go:build darwin

package target

import (
	"fmt"
	"os/exec"
	"sort"
	"strconv"
	"strings"
)

func ResolvePort(port int) ([]int, error) {
	pidSet := make(map[int]bool)

	// Query TCP listeners: lsof -i TCP:<port> -s TCP:LISTEN -n -P -t
	if out, err := exec.Command("lsof", "-i", fmt.Sprintf("TCP:%d", port), "-s", "TCP:LISTEN", "-n", "-P", "-t").Output(); err == nil {
		for _, pidStr := range strings.Split(strings.TrimSpace(string(out)), "\n") {
			if pid, err := strconv.Atoi(strings.TrimSpace(pidStr)); err == nil && pid > 0 {
				pidSet[pid] = true
			}
		}
	}

	// Query UDP bound sockets: lsof -i UDP:<port> -n -P -t
	// UDP is connectionless so there is no LISTEN state to filter on.
	if out, err := exec.Command("lsof", "-i", fmt.Sprintf("UDP:%d", port), "-n", "-P", "-t").Output(); err == nil {
		for _, pidStr := range strings.Split(strings.TrimSpace(string(out)), "\n") {
			if pid, err := strconv.Atoi(strings.TrimSpace(pidStr)); err == nil && pid > 0 {
				pidSet[pid] = true
			}
		}
	}

	if len(pidSet) == 0 {
		// Fallback: include processes with connected (non-listening) sockets
		// on this port — `lsof -i :port` matches either side of the connection.
		if out, err := exec.Command("lsof", "-i", fmt.Sprintf(":%d", port), "-n", "-P", "-t").Output(); err == nil {
			for _, pidStr := range strings.Split(strings.TrimSpace(string(out)), "\n") {
				if pid, err := strconv.Atoi(strings.TrimSpace(pidStr)); err == nil && pid > 0 {
					pidSet[pid] = true
				}
			}
		}
		if len(pidSet) == 0 {
			// Try alternative: netstat fallback
			return resolvePortNetstat(port)
		}
	}

	// collect all owning pids so callers can handle multi-owner sockets
	result := make([]int, 0, len(pidSet))
	for pid := range pidSet {
		result = append(result, pid)
	}
	sort.Ints(result)

	return result, nil
}

func resolvePortNetstat(port int) ([]int, error) {
	pidSet := make(map[int]bool)
	fallbackSet := make(map[int]bool)
	sawListenNoOwner := false
	portStr := fmt.Sprintf(".%d", port)

	// Check TCP: netstat -anv -p tcp
	if out, err := exec.Command("netstat", "-anv", "-p", "tcp").Output(); err == nil {
		for line := range strings.Lines(string(out)) {
			fields := strings.Fields(line)
			if len(fields) < 9 {
				continue
			}
			// Local address may be in field 3 (IPv4) or 4 (IPv6 with -v); be lenient.
			if !strings.HasSuffix(fields[3], portStr) && !(len(fields) > 4 && strings.HasSuffix(fields[4], portStr)) {
				continue
			}
			isListen := strings.Contains(line, "LISTEN")
			pid, err := strconv.Atoi(fields[8])
			if err != nil || pid <= 0 {
				if isListen {
					sawListenNoOwner = true
				}
				continue
			}
			if isListen {
				pidSet[pid] = true
			} else {
				fallbackSet[pid] = true
			}
		}
	}

	// Check UDP bound/connected sockets: netstat -anv -p udp (no LISTEN state).
	if out, err := exec.Command("netstat", "-anv", "-p", "udp").Output(); err == nil {
		for line := range strings.Lines(string(out)) {
			fields := strings.Fields(line)
			if len(fields) >= 9 && strings.HasSuffix(fields[3], portStr) {
				if pid, err := strconv.Atoi(fields[8]); err == nil && pid > 0 {
					pidSet[pid] = true
				}
			}
		}
	}

	if len(pidSet) == 0 && len(fallbackSet) > 0 {
		pidSet = fallbackSet
	}

	result := make([]int, 0, len(pidSet))
	for pid := range pidSet {
		result = append(result, pid)
	}
	sort.Ints(result)
	if len(result) > 0 {
		return result, nil
	}
	if sawListenNoOwner {
		return nil, ErrSocketOwnerUnknown
	}

	return nil, fmt.Errorf("no process listening on port %d", port)
}
