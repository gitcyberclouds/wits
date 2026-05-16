//go:build freebsd

package proc

import (
	"os/exec"
	"strconv"
	"strings"

	"github.com/pranshuparmar/witr/pkg/model"
)

// ListLockedFiles returns file locks observable via `fstat`. FreeBSD doesn't
// expose a clean system-wide lock table the way /proc/locks does, so this is
// best-effort: lines whose path or fd flags indicate a lock are emitted.
func ListLockedFiles() []*model.LockedFile {
	out, err := exec.Command("fstat").Output()
	if err != nil {
		return nil
	}

	var locks []*model.LockedFile
	for _, line := range strings.Split(string(out), "\n") {
		fields := strings.Fields(line)
		if len(fields) < 3 {
			continue
		}
		// USER CMD PID FD MOUNT INUM MODE SZ|DV R/W [NAME]
		if !lockIndicators(line) {
			continue
		}
		pid, err := strconv.Atoi(fields[2])
		if err != nil || pid <= 0 {
			continue
		}
		mode := "RW"
		if len(fields) >= 9 {
			mode = strings.ToUpper(fields[8])
		}
		path := ""
		if len(fields) >= 10 {
			path = strings.Join(fields[9:], " ")
		}
		locks = append(locks, &model.LockedFile{
			PID:     pid,
			Process: fields[1],
			Path:    path,
			Type:    "FLOCK",
			Mode:    mode,
		})
	}
	return locks
}

func lockIndicators(line string) bool {
	return strings.Contains(line, ".lock") ||
		strings.Contains(line, "LOCK") ||
		strings.Contains(line, "/lock")
}
