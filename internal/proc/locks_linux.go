//go:build linux

package proc

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"syscall"

	"github.com/pranshuparmar/witr/pkg/model"
)

// statKey returns the file's inode as a decimal string. /proc/locks emits
// device:inode but the device-numbering format the kernel uses there doesn't
// always match what userspace stat returns, so we match on inode alone.
// Inode collisions across filesystems are theoretically possible but
// vanishingly rare in practice.
func statKey(info os.FileInfo) string {
	st, ok := info.Sys().(*syscall.Stat_t)
	if !ok {
		return ""
	}
	return fmt.Sprintf("%d", st.Ino)
}

// ListLockedFiles returns every file lock currently held on the system,
// parsed from /proc/locks. Inodes are resolved to paths by scanning the
// owning process's /proc/<pid>/fd/* — incomplete coverage is acceptable
// (anonymous fds, vanished processes, etc. just get the device:inode literal).
func ListLockedFiles() []*model.LockedFile {
	data, err := os.ReadFile("/proc/locks")
	if err != nil {
		return nil
	}

	pathCache := make(map[int]map[string]string)
	commCache := make(map[int]string)

	var out []*model.LockedFile
	for _, line := range strings.Split(string(data), "\n") {
		fields := strings.Fields(line)
		// /proc/locks: <id>: <type> <kind> <access> <pid> <maj:min:inode> <start> <end>
		if len(fields) < 8 {
			continue
		}
		lockType := fields[1]
		access := fields[3]
		pidStr := fields[4]
		devInode := fields[5]

		pid, err := strconv.Atoi(pidStr)
		if err != nil || pid <= 0 {
			continue
		}

		path := resolveLockPath(pid, devInode, pathCache)
		if path == "" {
			path = devInode
		}

		out = append(out, &model.LockedFile{
			PID:     pid,
			Process: lockProcessName(pid, commCache),
			Path:    path,
			Type:    lockType,
			Mode:    access,
		})
	}
	return out
}

// resolveLockPath walks /proc/<pid>/fd to map a /proc/locks device:inode
// entry back to a file path. Cached per-PID across all locks in this scan.
// Matches on inode (last segment of the device:inode key) for portability.
func resolveLockPath(pid int, devInode string, cache map[int]map[string]string) string {
	inodeKey := devInode
	if i := strings.LastIndex(inodeKey, ":"); i >= 0 {
		inodeKey = inodeKey[i+1:]
	}

	if m, ok := cache[pid]; ok {
		return m[inodeKey]
	}

	m := make(map[string]string)
	cache[pid] = m

	fdDir := fmt.Sprintf("/proc/%d/fd", pid)
	entries, err := os.ReadDir(fdDir)
	if err != nil {
		return ""
	}
	for _, entry := range entries {
		target, err := os.Readlink(fdDir + "/" + entry.Name())
		if err != nil {
			continue
		}
		info, err := os.Stat(target)
		if err != nil {
			continue
		}
		if key := statKey(info); key != "" {
			m[key] = target
		}
	}
	return m[inodeKey]
}

func lockProcessName(pid int, cache map[int]string) string {
	if name, ok := cache[pid]; ok {
		return name
	}
	data, err := os.ReadFile(fmt.Sprintf("/proc/%d/comm", pid))
	if err != nil {
		cache[pid] = ""
		return ""
	}
	name := strings.TrimSpace(string(data))
	cache[pid] = name
	return name
}
