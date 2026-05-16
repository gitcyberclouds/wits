//go:build freebsd

package proc

import (
	"os/exec"
	"strconv"
	"strings"

	"github.com/pranshuparmar/witr/pkg/model"
)

// ListAllOpenFiles uses `fstat` to enumerate open files across all processes.
// Non-file entries (pipes, sockets, kqueue) and obvious noise are dropped so
// the result resembles "files a user would recognize on disk".
func ListAllOpenFiles() []*model.LockedFile {
	out, err := exec.Command("fstat").Output()
	if err != nil {
		return nil
	}

	var files []*model.LockedFile
	for _, line := range strings.Split(string(out), "\n") {
		fields := strings.Fields(line)
		// USER CMD PID FD MOUNT INUM MODE SZ|DV R/W [NAME]
		if len(fields) < 10 {
			continue
		}
		pid, err := strconv.Atoi(fields[2])
		if err != nil || pid <= 0 {
			continue
		}
		path := strings.Join(fields[9:], " ")
		if !isInterestingFreebsdPath(path) {
			continue
		}
		mode := strings.ToUpper(fields[8])
		files = append(files, &model.LockedFile{
			PID:     pid,
			Process: fields[1],
			Path:    path,
			Type:    "OPEN",
			Mode:    mode,
		})
	}
	return files
}

func isInterestingFreebsdPath(p string) bool {
	if p == "" || p == "-" || p == "/dev/null" {
		return false
	}
	if strings.HasPrefix(p, "/dev/tty") || strings.HasPrefix(p, "/dev/pts/") {
		return false
	}
	return true
}
