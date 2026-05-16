//go:build linux

package proc

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/pranshuparmar/witr/pkg/model"
)

// ListAllOpenFiles walks /proc/<pid>/fd for every visible process and
// returns one LockedFile entry per open fd that resolves to a real file
// path. Kernel-internal fds (sockets, pipes, anon_inodes, /proc, /sys,
// /dev/null, /memfd) are dropped — they're virtually never what a user is
// searching for and dominate the raw count.
//
// Type is set to "OPEN" and Mode is derived from /proc/<pid>/fdinfo flags
// when readable; missing fdinfo just leaves Mode empty.
func ListAllOpenFiles() []*model.LockedFile {
	procDirs, err := os.ReadDir("/proc")
	if err != nil {
		return nil
	}

	commCache := make(map[int]string)
	var out []*model.LockedFile

	for _, d := range procDirs {
		if !d.IsDir() {
			continue
		}
		pid, err := strconv.Atoi(d.Name())
		if err != nil || pid <= 0 {
			continue
		}

		fdDir := fmt.Sprintf("/proc/%d/fd", pid)
		entries, err := os.ReadDir(fdDir)
		if err != nil {
			continue // permission denied or process gone
		}

		for _, fd := range entries {
			target, err := os.Readlink(fdDir + "/" + fd.Name())
			if err != nil {
				continue
			}
			if !isInterestingFile(target) {
				continue
			}
			out = append(out, &model.LockedFile{
				PID:     pid,
				Process: lockProcessName(pid, commCache),
				Path:    target,
				Type:    "OPEN",
				Mode:    fdMode(pid, fd.Name()),
			})
		}
	}
	return out
}

// isInterestingFile returns false for fd link targets that are virtually
// never what users are looking for when asking "who has this file open?".
func isInterestingFile(target string) bool {
	switch {
	case strings.HasPrefix(target, "socket:["):
	case strings.HasPrefix(target, "pipe:["):
	case strings.HasPrefix(target, "anon_inode:"):
	case strings.HasPrefix(target, "/memfd:"):
	case strings.HasPrefix(target, "/proc/"):
	case strings.HasPrefix(target, "/sys/"):
	case target == "/dev/null":
	case strings.HasPrefix(target, "/dev/tty"):
	case strings.HasPrefix(target, "/dev/pts/"):
	default:
		return true
	}
	return false
}

// fdMode reads /proc/<pid>/fdinfo/<fd> for the O_ACCMODE flag bits and
// returns "R", "W", or "RW". Returns "" if the file isn't readable.
func fdMode(pid int, fd string) string {
	data, err := os.ReadFile(fmt.Sprintf("/proc/%d/fdinfo/%s", pid, fd))
	if err != nil {
		return ""
	}
	for _, line := range strings.Split(string(data), "\n") {
		if !strings.HasPrefix(line, "flags:") {
			continue
		}
		flags, err := strconv.ParseInt(strings.TrimSpace(strings.TrimPrefix(line, "flags:")), 8, 64)
		if err != nil {
			return ""
		}
		switch flags & 3 { // O_ACCMODE = 3; 0=RDONLY, 1=WRONLY, 2=RDWR
		case 0:
			return "R"
		case 1:
			return "W"
		case 2:
			return "RW"
		}
	}
	return ""
}
