//go:build darwin

package proc

import (
	"os/exec"
	"strconv"
	"strings"

	"github.com/pranshuparmar/witr/pkg/model"
)

// ListLockedFiles uses `lsof -l` to surface file locks. macOS has no
// /proc/locks equivalent; this is best-effort coverage and may take a
// noticeable beat on busy systems because lsof scans every open fd.
func ListLockedFiles() []*model.LockedFile {
	// lsof may exit non-zero when it can't read one of the processes it
	// scans (permission denied, process exiting mid-scan, etc.) while still
	// emitting valid rows on stdout for the rest; salvage stdout when present.
	out, err := exec.Command("lsof", "-l", "-n", "-P").Output()
	if err != nil && len(out) == 0 {
		return nil
	}

	var locks []*model.LockedFile
	for _, line := range strings.Split(string(out), "\n") {
		fields := strings.Fields(line)
		// COMMAND PID USER FD TYPE DEVICE SIZE/OFF NODE NAME
		if len(fields) < 9 {
			continue
		}
		// lsof prints the lock flag inside the FD column, e.g. "5uW" (W = write lock).
		fd := fields[3]
		mode := lsofFDLockMode(fd)
		if mode == "" {
			continue
		}
		pid, err := strconv.Atoi(fields[1])
		if err != nil || pid <= 0 {
			continue
		}
		path := strings.Join(fields[8:], " ")
		locks = append(locks, &model.LockedFile{
			PID:     pid,
			Process: fields[0],
			Path:    path,
			Type:    "FLOCK",
			Mode:    mode,
		})
	}
	return locks
}

// lsofFDLockMode extracts the lock indicator from an lsof FD column.
// Per lsof(8): trailing 'W' = write lock, 'R' = read lock, 'r'/'w' = locked
// region, 'u' = upgradable. Returns "" when no lock flag is present.
func lsofFDLockMode(fd string) string {
	if fd == "" {
		return ""
	}
	switch fd[len(fd)-1] {
	case 'W', 'w':
		return "WRITE"
	case 'R', 'r':
		return "READ"
	case 'u', 'U':
		return "RW"
	}
	return ""
}
