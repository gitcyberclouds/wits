//go:build freebsd

package target

import (
	"fmt"
	"os/exec"
	"strconv"
	"strings"
)

func ResolvePort(port int) ([]int, error) {
	// Use sockstat to find the process listening on this port
	// sockstat -4 -l -P tcp -p <port>
	// sockstat -6 -l -P tcp -p <port>
	pidSet := make(map[int]bool)

	for _, flag := range []string{"-4", "-6"} {
		out, err := exec.Command("sockstat", flag, "-l", "-P", "tcp", "-p", strconv.Itoa(port)).Output()
		if err != nil {
			continue
		}

		// Parse sockstat output
		// USER     COMMAND    PID   FD PROTO  LOCAL ADDRESS         FOREIGN ADDRESS
		// root     nginx      1234  6  tcp4   *:80                  *:*
		for line := range strings.Lines(string(out)) {
			fields := strings.Fields(line)
			if len(fields) < 6 {
				continue
			}

			// Skip header
			if fields[0] == "USER" {
				continue
			}

			pid, err := strconv.Atoi(fields[2])
			if err == nil && pid > 0 {
				pidSet[pid] = true
			}
		}
	}

	if len(pidSet) == 0 {
		// Try netstat as fallback
		return resolvePortNetstat(port)
	}

	// Return the lowest PID (the main listener, not forked children)
	var result []int
	minPID := 0
	for pid := range pidSet {
		if minPID == 0 || pid < minPID {
			minPID = pid
		}
	}
	if minPID > 0 {
		result = append(result, minPID)
	}

	if len(result) == 0 {
		return nil, fmt.Errorf("socket found but owning process not detected")
	}

	return result, nil
}

func resolvePortNetstat(port int) ([]int, error) {
	// Fallback using netstat
	// On FreeBSD: netstat -an -p tcp | grep LISTEN
	out, err := exec.Command("netstat", "-an", "-p", "tcp").Output()
	if err != nil {
		return nil, fmt.Errorf("no process listening on port %d", port)
	}

	portStr := fmt.Sprintf(".%d", port)
	portColonStr := fmt.Sprintf(":%d", port)

	for line := range strings.Lines(string(out)) {
		if !strings.Contains(line, "LISTEN") {
			continue
		}
		if !strings.Contains(line, portStr) && !strings.Contains(line, portColonStr) {
			continue
		}

		// Unfortunately, basic netstat doesn't show PID on FreeBSD
		// We need to use sockstat or fstat for that
		// Try fstat as last resort
		return resolvePortFstat(port)
	}

	return nil, fmt.Errorf("no process listening on port %d", port)
}

func resolvePortFstat(port int) ([]int, error) {
	// Use fstat to find processes with open sockets
	// This is less efficient but works as a fallback
	out, err := exec.Command("fstat").Output()
	if err != nil {
		return nil, fmt.Errorf("no process listening on port %d", port)
	}

	portStr := fmt.Sprintf(":%d", port)
	pidSet := make(map[int]bool)

	for line := range strings.Lines(string(out)) {
		if !strings.Contains(line, "tcp") {
			continue
		}
		if !strings.Contains(line, portStr) {
			continue
		}

		fields := strings.Fields(line)
		if len(fields) >= 3 {
			pid, err := strconv.Atoi(fields[2])
			if err == nil && pid > 0 {
				pidSet[pid] = true
			}
		}
	}

	if len(pidSet) == 0 {
		return nil, fmt.Errorf("no process listening on port %d", port)
	}

	// Return the lowest PID
	var result []int
	minPID := 0
	for pid := range pidSet {
		if minPID == 0 || pid < minPID {
			minPID = pid
		}
	}
	if minPID > 0 {
		result = append(result, minPID)
	}

	return result, nil
}
