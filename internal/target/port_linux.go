//go:build linux

package target

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

func findSocketInodes(port int) (map[string]bool, error) {
	inodes := make(map[string]bool)

	files := []string{"/proc/net/tcp", "/proc/net/tcp6"}
	targetHex := fmt.Sprintf("%04X", port)

	for _, file := range files {
		data, err := os.ReadFile(file)
		if err != nil {
			continue
		}

		lines := strings.Split(string(data), "\n")
		for _, line := range lines[1:] {
			fields := strings.Fields(line)
			if len(fields) < 10 {
				continue
			}

			localAddr := fields[1]
			parts := strings.Split(localAddr, ":")
			if len(parts) != 2 {
				continue
			}

			if parts[1] == targetHex {
				inodes[fields[9]] = true
			}
		}
	}

	if len(inodes) == 0 {
		return nil, fmt.Errorf("no process listening on port %d", port)
	}

	return inodes, nil
}

func ResolvePort(port int) ([]int, error) {
	inodes, err := findSocketInodes(port)
	if err != nil {
		return nil, err
	}

	// Map inode to PID that owns the LISTEN socket
	pidSet := make(map[int]bool)
	procEntries, _ := os.ReadDir("/proc")
	for _, entry := range procEntries {
		pid, err := strconv.Atoi(entry.Name())
		if err != nil {
			continue
		}

		fdDir := filepath.Join("/proc", entry.Name(), "fd")
		fds, err := os.ReadDir(fdDir)
		if err != nil {
			continue
		}

		for _, fd := range fds {
			link, err := os.Readlink(filepath.Join(fdDir, fd.Name()))
			if err != nil {
				continue
			}

			if strings.HasPrefix(link, "socket:[") {
				inode := strings.TrimSuffix(strings.TrimPrefix(link, "socket:["), "]")
				if inodes[inode] {
					// Only add the first PID that owns the LISTEN socket
					pidSet[pid] = true
				}
			}
		}
	}

	// Only return the lowest PID (the main listener, not forked children)
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
