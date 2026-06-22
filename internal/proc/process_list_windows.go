//go:build windows

package proc

import (
	"github.com/pranshuparmar/witr/pkg/model"
)

// ListProcesses returns all running processes with the columns the TUI renders:
// PID/PPID/command name, owner, start time, command line, CPU% and resident
// memory. Each process is opened to read its metrics; fields that can't be read
// (protected/system processes without elevation) are left empty rather than
// failing the whole list. The TUI loads this asynchronously.
//
// The command line is read via NtQueryInformationProcess (windowsProcessCmdline)
// rather than a PEB walk, so it performs no remote process-memory access and is
// safe to call across every process.
func ListProcesses() ([]model.Process, error) {
	procs, err := enumerateProcesses()
	if err != nil {
		return nil, err
	}

	out := make([]model.Process, 0, len(procs))
	for _, p := range procs {
		rss, cpu, cpuTime, started := windowsProcMetrics(p.PID)
		out = append(out, model.Process{
			PID:           p.PID,
			PPID:          p.PPID,
			Command:       p.Exe,
			Cmdline:       windowsProcessCmdline(p.PID),
			User:          readUser(p.PID),
			StartedAt:     started,
			CPUPercent:    cpu,
			MemoryRSS:     rss,
			MemoryPercent: windowsMemoryPercent(rss),
			Health:        windowsHealth(rss, cpuTime),
		})
	}
	return out, nil
}

// ListProcessSnapshot collects a lightweight view of running processes for
// child/descendant discovery. Backed by ToolHelp32 (no PowerShell, no WMI) so
// it never blocks on a stalled CIM provider.
func ListProcessSnapshot() ([]model.Process, error) {
	procs, err := enumerateProcesses()
	if err != nil {
		return nil, err
	}
	out := make([]model.Process, 0, len(procs))
	for _, p := range procs {
		out = append(out, model.Process{
			PID:     p.PID,
			PPID:    p.PPID,
			Command: p.Exe,
		})
	}
	return out, nil
}
