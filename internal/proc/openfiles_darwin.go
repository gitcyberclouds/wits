//go:build darwin

package proc

import (
	"os/exec"
	"strconv"
	"strings"

	"github.com/pranshuparmar/witr/pkg/model"
)

// ListAllOpenFiles uses `lsof` to enumerate open files across all processes.
// Kernel-internal entries (sockets, pipes, kqueue, etc.) are filtered out so
// the result roughly resembles "files a user would recognize on disk". This
// can take a noticeable beat on busy systems.
func ListAllOpenFiles() []*model.LockedFile {
	// lsof may exit non-zero when it can't read one of the processes it
	// scans (permission denied, process exiting mid-scan, etc.) while still
	// emitting valid rows on stdout for the rest; salvage stdout when present.
	out, err := exec.Command("lsof", "-l", "-n", "-P").Output()
	if err != nil && len(out) == 0 {
		return nil
	}

	var files []*model.LockedFile
	for _, line := range strings.Split(string(out), "\n") {
		fields := strings.Fields(line)
		// COMMAND PID USER FD TYPE DEVICE SIZE/OFF NODE NAME
		if len(fields) < 9 {
			continue
		}
		fdType := fields[4]
		if !lsofTypeIsFile(fdType) {
			continue
		}
		pid, err := strconv.Atoi(fields[1])
		if err != nil || pid <= 0 {
			continue
		}
		path := strings.Join(fields[8:], " ")
		if !isInterestingDarwinPath(path) {
			continue
		}
		files = append(files, &model.LockedFile{
			PID:     pid,
			Process: fields[0],
			Path:    path,
			Type:    "OPEN",
			Mode:    lsofFDMode(fields[3]),
		})
	}
	return files
}

// lsofTypeIsFile keeps regular files and directories; drops sockets, pipes,
// kqueue, character/block devices, and other kernel-internal handle types.
func lsofTypeIsFile(t string) bool {
	switch t {
	case "REG", "DIR":
		return true
	}
	return false
}

// isInterestingDarwinPath drops paths that are almost never useful when a
// user is asking "who has this file open?".
func isInterestingDarwinPath(p string) bool {
	if p == "" || p == "/dev/null" {
		return false
	}
	if strings.HasPrefix(p, "/dev/tty") || strings.HasPrefix(p, "/dev/ttys") {
		return false
	}
	return true
}

// lsofFDMode strips the lock indicators from an lsof FD column and returns
// R/W/RW based on the access mode character. Returns "" if unrecognized.
func lsofFDMode(fd string) string {
	if fd == "" {
		return ""
	}
	// Strip trailing lock indicators (W,R,r,w,u,U).
	end := len(fd)
	for end > 0 {
		switch fd[end-1] {
		case 'W', 'R', 'r', 'w', 'u', 'U', 'N', ' ':
			end--
			continue
		}
		break
	}
	if end == 0 {
		return ""
	}
	switch fd[end-1] {
	case 'r':
		return "R"
	case 'w':
		return "W"
	case 'u':
		return "RW"
	}
	return ""
}
