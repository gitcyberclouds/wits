//go:build linux

package proc

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/pranshuparmar/witr/pkg/model"
)

// ListProcesses returns all running processes with the columns the TUI renders
// (PID, PPID, command, user, start time, CPU%, RSS, mem%, command line). It
// reads /proc directly instead of forking `ps -axo`, computing the same
// lifetime-average CPU% that ps reports — no subprocess per refresh.
func ListProcesses() ([]model.Process, error) {
	entries, err := os.ReadDir("/proc")
	if err != nil {
		return nil, fmt.Errorf("read /proc: %w", err)
	}

	// Per-list invariants, computed once rather than per process.
	ticks := ticksPerSecond()
	boot := bootTime()
	totalMem := float64(totalMemoryBytes())
	pageSize := float64(os.Getpagesize())

	processes := make([]model.Process, 0, len(entries))
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		pid, err := strconv.Atoi(entry.Name())
		if err != nil {
			continue
		}
		if p, ok := readProcessListEntry(pid, ticks, boot, totalMem, pageSize); ok {
			processes = append(processes, p)
		}
	}
	return processes, nil
}

// readProcessListEntry reads the TUI list columns for one PID from /proc,
// mirroring the fields the old `ps -axo` invocation produced. Returns ok=false
// when the process vanished mid-read or its stat is malformed.
func readProcessListEntry(pid, ticks int, boot time.Time, totalMem, pageSize float64) (model.Process, bool) {
	stat, err := os.ReadFile(fmt.Sprintf("/proc/%d/stat", pid))
	if err != nil {
		return model.Process{}, false
	}
	raw := string(stat)
	open := strings.Index(raw, "(")
	closeParen := strings.LastIndex(raw, ")")
	if open == -1 || closeParen == -1 || closeParen+2 >= len(raw) {
		return model.Process{}, false
	}
	comm := raw[open+1 : closeParen]
	fields := strings.Fields(raw[closeParen+2:])
	if len(fields) < 22 {
		return model.Process{}, false
	}

	ppid, _ := strconv.Atoi(fields[1])
	utime, _ := strconv.ParseFloat(fields[11], 64)
	stime, _ := strconv.ParseFloat(fields[12], 64)
	startTicks, _ := strconv.ParseInt(fields[19], 10, 64)
	rssPages, _ := strconv.ParseFloat(fields[21], 64)

	startedAt := startTimeFromTicks(boot, startTicks, ticks)
	memBytes := rssPages * pageSize

	// Lifetime-average CPU%: total CPU time over wall-clock since start (what ps reports).
	cpuPercent := 0.0
	if wall := time.Since(startedAt).Seconds(); wall > 0 {
		cpuPercent = (utime + stime) / float64(ticks) / wall * 100.0
	}
	memPercent := 0.0
	if totalMem > 0 {
		memPercent = memBytes / totalMem * 100.0
	}

	cmdline := ""
	if b, err := os.ReadFile(fmt.Sprintf("/proc/%d/cmdline", pid)); err == nil {
		cmdline = strings.TrimSpace(strings.ReplaceAll(string(b), "\x00", " "))
	}
	displayName := deriveDisplayCommand(comm, cmdline)
	if displayName == "" {
		displayName = comm
	}
	if cmdline == "" {
		cmdline = displayName
	}

	return model.Process{
		PID:           pid,
		PPID:          ppid,
		Command:       displayName,
		Cmdline:       cmdline,
		User:          readUser(pid),
		StartedAt:     startedAt,
		CPUPercent:    cpuPercent,
		MemoryRSS:     uint64(memBytes),
		MemoryPercent: memPercent,
	}, true
}

// ListProcessSnapshot collects a lightweight view of running processes
// for child/descendant discovery. We avoid full ReadProcess calls to keep
// this path fast and to reduce permission-sensitive reads.
func ListProcessSnapshot() ([]model.Process, error) {
	entries, err := os.ReadDir("/proc")
	if err != nil {
		return nil, fmt.Errorf("read /proc: %w", err)
	}

	processes := make([]model.Process, 0, len(entries))
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		pid, err := strconv.Atoi(entry.Name())
		if err != nil {
			continue
		}

		statPath := fmt.Sprintf("/proc/%d/stat", pid)
		stat, err := os.ReadFile(statPath)
		if err != nil {
			continue
		}

		proc, err := parseStatSnapshot(pid, stat)
		if err != nil {
			continue
		}

		processes = append(processes, proc)
	}

	return processes, nil
}

func parseStatSnapshot(pid int, stat []byte) (model.Process, error) {
	raw := string(stat)
	open := strings.Index(raw, "(")
	close := strings.LastIndex(raw, ")")
	// close+2 >= len(raw) guards the raw[close+2:] slice below: a stat ending at
	// the comm's ')' would otherwise panic. Matches ReadProcess's bounds check.
	if open == -1 || close == -1 || close <= open || close+2 >= len(raw) {
		return model.Process{}, fmt.Errorf("invalid stat format")
	}

	comm := raw[open+1 : close]
	fields := strings.Fields(raw[close+2:])
	if len(fields) < 2 {
		return model.Process{}, fmt.Errorf("invalid stat format")
	}

	ppid, err := strconv.Atoi(fields[1])
	if err != nil {
		return model.Process{}, fmt.Errorf("invalid ppid")
	}

	return model.Process{
		PID:     pid,
		PPID:    ppid,
		Command: comm,
	}, nil
}
