package process

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/pranshuparmar/witr/pkg/model"
)

const clockTicks = 100 // safe default, good enough for now

func readStat(pid int) (model.Process, error) {
	data, err := os.ReadFile(fmt.Sprintf("/proc/%d/stat", pid))
	if err != nil {
		return model.Process{}, err
	}

	// stat format is tricky because comm is in parentheses
	// pid (comm) state ppid ...
	content := string(data)

	start := strings.Index(content, "(")
	end := strings.LastIndex(content, ")")
	if start == -1 || end == -1 || end <= start {
		return model.Process{}, fmt.Errorf("invalid stat format")
	}

	comm := content[start+1 : end]
	fields := strings.Fields(content[end+2:])

	if len(fields) < 20 {
		return model.Process{}, fmt.Errorf("stat fields missing")
	}

	ppid, _ := strconv.Atoi(fields[1])
	startTicks, _ := strconv.ParseInt(fields[19], 10, 64)

	startTime := bootTime().Add(time.Duration(startTicks/clockTicks) * time.Second)

	return model.Process{
		PID:       pid,
		PPID:      ppid,
		Command:   comm,
		StartTime: startTime,
	}, nil
}

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

		proc, err := readStat(current)
		if err != nil {
			break
		}

		chain = append([]model.Process{proc}, chain...)

		if proc.PPID == 0 || proc.PID == 1 {
			break
		}
		current = proc.PPID
	}

	if len(chain) == 0 {
		return nil, fmt.Errorf("no process ancestry found")
	}

	return chain, nil
}
