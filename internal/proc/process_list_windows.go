//go:build windows

package proc

import (
	"github.com/pranshuparmar/witr/pkg/model"
)

// ListProcesses returns a list of all running processes with basic details
// (PID, PPID, command). Used by the TUI to populate the process list.
func ListProcesses() ([]model.Process, error) {
	return ListProcessSnapshot()
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
