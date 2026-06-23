package target

import (
	"fmt"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
)

func ResolveFile(path string) ([]int, error) {
	// Absolute path so a leading-dash path can't be parsed as an fstat option.
	absPath, err := filepath.Abs(path)
	if err != nil {
		absPath = path
	}

	cmd := exec.Command("fstat", absPath)
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("fstat failed: %w", err)
	}

	var pids []int
	lines := strings.Split(string(out), "\n")
	for i, line := range lines {
		if i == 0 {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) >= 3 {
			pidStr := fields[2]
			pid, err := strconv.Atoi(pidStr)
			if err == nil {
				pids = append(pids, pid)
			}
		}
	}

	if len(pids) == 0 {
		return nil, fmt.Errorf("no process found holding file: %s", path)
	}
	return pids, nil
}
