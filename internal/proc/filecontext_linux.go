//go:build linux

package proc

import (
	"fmt"
	"os"
	"sort"
	"strconv"
	"strings"
	"syscall"

	"golang.org/x/sys/unix"

	"github.com/pranshuparmar/witr/pkg/model"
)

// GetFileContext returns file descriptor and lock info for a process
// Will return nil if the context could not be gathered.
func GetFileContext(pid int) *model.FileContext {
	var fileContext model.FileContext

	fdDir := fmt.Sprintf("/proc/%v/fd", pid)
	fdFiles, err := os.ReadDir(fdDir)
	if err != nil {
		return nil
	}

	fileContext.OpenFiles = len(fdFiles)
	fileContext.FileLimit = getFileLimit(pid)
	fileContext.LockedFiles = getLockedFiles(pid)

	return &fileContext
}

func getFileLimit(pid int) int {
	var linuxDefaultMaxOpenFile = getDefaultMaxOpenFiles()

	// Read /proc/<pid>/limits for file limit
	data, err := os.ReadFile(fmt.Sprintf("/proc/%v/limits", pid))
	if err != nil {
		return linuxDefaultMaxOpenFile
	}

	dataString := string(data)
	for line := range strings.Lines(dataString) {
		if !strings.HasPrefix(line, "Max open files") {
			continue
		}

		// Data in format: "Max open files $SOFT_LOCK_NUMBER $HARD_LOCK_NUMBER files"
		fields := strings.Fields(line)
		if len(fields) < 4 {
			return linuxDefaultMaxOpenFile
		}
		softLimitString := fields[3]

		if softLimitString == "unlimited" {
			return 0
		}

		softLimit, err := strconv.Atoi(softLimitString)
		if err != nil {
			return linuxDefaultMaxOpenFile
		}

		return softLimit
	}

	return linuxDefaultMaxOpenFile
}

func getDefaultMaxOpenFiles() int {
	// This seems to be a common default for many systems.
	const reasonableDefault int = 1024

	// https://www.man7.org/linux/man-pages/man2/getrlimit.2.html
	var rlimit syscall.Rlimit
	err := syscall.Getrlimit(syscall.RLIMIT_NOFILE, &rlimit)
	if err != nil {
		return reasonableDefault
	}

	return int(rlimit.Max)
}

// getLockedFiles returns the files locked by the process, read from
// /proc/locks. Paths are resolved by matching each lock's device:inode against
// the process's own open fds, keeping the work bounded to this one process.
// (lslocks resolves every lock on the system and blocks on slow mounts.)
func getLockedFiles(pid int) []string {
	data, err := os.ReadFile("/proc/locks")
	if err != nil {
		return nil
	}

	// /proc/locks line: "<id>: <TYPE> <KIND> <ACCESS> <PID> <MAJOR:MINOR:INODE>
	// <START> <END>", with MAJOR:MINOR in hex and INODE in decimal.
	type fileID struct{ dev, ino uint64 }
	ids := map[fileID]string{} // -> raw "major:minor:inode" identifier for fallback
	pidStr := strconv.Itoa(pid)
	for _, line := range strings.Split(string(data), "\n") {
		fields := strings.Fields(line)
		if len(fields) < 8 || fields[4] != pidStr {
			continue
		}
		parts := strings.Split(fields[5], ":")
		if len(parts) != 3 {
			continue
		}
		major, err1 := strconv.ParseUint(parts[0], 16, 32)
		minor, err2 := strconv.ParseUint(parts[1], 16, 32)
		ino, err3 := strconv.ParseUint(parts[2], 10, 64)
		if err1 != nil || err2 != nil || err3 != nil {
			continue
		}
		ids[fileID{unix.Mkdev(uint32(major), uint32(minor)), ino}] = fields[5]
	}
	if len(ids) == 0 {
		return nil
	}

	// Resolve paths from the process's own fds; keep the raw identifier for any
	// lock that can't be matched to an open fd (e.g. an unlinked file).
	paths := map[fileID]string{}
	fdDir := fmt.Sprintf("/proc/%d/fd", pid)
	if entries, err := os.ReadDir(fdDir); err == nil {
		for _, e := range entries {
			if len(paths) == len(ids) {
				break
			}
			fdPath := fdDir + "/" + e.Name()
			info, err := os.Stat(fdPath)
			if err != nil {
				continue
			}
			st, ok := info.Sys().(*syscall.Stat_t)
			if !ok {
				continue
			}
			id := fileID{uint64(st.Dev), uint64(st.Ino)}
			if _, want := ids[id]; want {
				if target, err := os.Readlink(fdPath); err == nil {
					paths[id] = target
				}
			}
		}
	}

	out := make([]string, 0, len(ids))
	for id, raw := range ids {
		if p := paths[id]; p != "" {
			out = append(out, p)
		} else {
			out = append(out, raw)
		}
	}
	sort.Strings(out)
	return out
}
