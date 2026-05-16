//go:build windows

package proc

import (
	"fmt"

	"github.com/pranshuparmar/witr/pkg/model"
)

// ResolveChildren returns the direct child processes for the provided PID.
// Backed by ToolHelp32 so it never blocks on a stalled WMI/CIM service.
func ResolveChildren(pid int) ([]model.Process, error) {
	if pid <= 0 {
		return nil, fmt.Errorf("invalid pid")
	}

	procs, err := enumerateProcesses()
	if err != nil {
		return nil, fmt.Errorf("enumerate processes: %w", err)
	}

	children := make([]model.Process, 0)
	for _, p := range procs {
		if p.PPID != pid {
			continue
		}
		children = append(children, model.Process{
			PID:     p.PID,
			PPID:    p.PPID,
			Command: p.Exe,
		})
	}
	sortProcesses(children)
	return children, nil
}
