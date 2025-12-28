package process

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	proc "github.com/pranshuparmar/witr/internal/linux/proc"
	"github.com/pranshuparmar/witr/pkg/model"
)

const clockTicks = 100 // safe default, good enough for now

func bootTime() time.Time {
	data, err := os.ReadFile("/proc/stat")
	if err != nil {
		return time.Now()
	}

	lines := strings.Split(string(data), "\n")
	for _, line := range lines {
		if strings.HasPrefix(line, "btime") {
			parts := strings.Fields(line)
			if len(parts) == 2 {
				sec, _ := strconv.ParseInt(parts[1], 10, 64)
				return time.Unix(sec, 0)
			}
		}
	}
	return time.Now()
}

func BuildAncestry(pid int) ([]model.Process, error) {
	var chain []model.Process
	seen := make(map[int]bool)

	current := pid

	for current > 0 {
		if seen[current] {
			break // loop protection
		}
		seen[current] = true

		p, err := proc.ReadProcess(current)
		if err != nil {
			break
		}

		chain = append([]model.Process{p}, chain...)

		if p.PPID == 0 || p.PID == 1 {
			break
		}
		current = p.PPID
	}

	if len(chain) == 0 {
		return nil, fmt.Errorf("no process ancestry found")
	}

	return chain, nil
}
