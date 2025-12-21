package target

import (
	"fmt"
	"os"
	"strconv"
	"strings"
)

func ResolveName(name string) ([]int, error) {
	var pids []int

	entries, _ := os.ReadDir("/proc")
	for _, e := range entries {
		pid, err := strconv.Atoi(e.Name())
		if err != nil {
			continue
		}

		comm, err := os.ReadFile("/proc/" + e.Name() + "/comm")
		if err != nil {
			continue
		}

		if strings.TrimSpace(string(comm)) == name {
			pids = append(pids, pid)
		}
	}

	if len(pids) == 0 {
		return nil, fmt.Errorf("no running process named %q", name)
	}

	return pids, nil
}
