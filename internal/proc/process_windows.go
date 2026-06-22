//go:build windows

package proc

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/pranshuparmar/witr/pkg/model"
)

func ReadProcess(pid int) (model.Process, error) {
	// PID 0 is the System Idle Process on Windows (and negative PIDs are never
	// valid), so reject them rather than returning the idle pseudo-process —
	// this matches the other platforms, where /proc/0 or `ps -p 0` fails.
	if pid <= 0 {
		return model.Process{}, fmt.Errorf("invalid pid %d", pid)
	}
	info, err := GetProcessDetailedInfo(pid)
	if err != nil {
		return model.Process{}, err
	}

	name := ""
	if info.Exe != "" {
		name = filepath.Base(info.Exe)
	}

	procSockets := GetSocketsForPID(pid)
	serviceName := detectWindowsServiceSource(pid)
	container := detectContainerFromCmdline(info.CommandLine)
	gitRepo, gitBranch := detectGitInfo(info.Cwd)

	// Resident memory (working set) and lifetime-average CPU%. CPU mirrors the
	// figure shown in the verbose report (ResourceContext) so every output mode
	// reports the same value.
	rss, cpu, cpuTime, _ := windowsProcMetrics(pid)

	return model.Process{
		PID:           pid,
		PPID:          info.PPID,
		Command:       name,
		Cmdline:       info.CommandLine,
		Exe:           info.Exe,
		StartedAt:     info.StartedAt,
		User:          readUser(pid),
		CPUPercent:    cpu,
		MemoryRSS:     rss,
		MemoryPercent: windowsMemoryPercent(rss),
		WorkingDir:    info.Cwd,
		GitRepo:       gitRepo,
		GitBranch:     gitBranch,
		Sockets:       procSockets,
		Health:        windowsHealth(rss, cpuTime),
		Forked:        "unknown",
		Env:           info.Env,
		Service:       serviceName,
		Container:     container,
		ExeDeleted:    isWindowsBinaryDeleted(info.Exe),
	}, nil
}

func isWindowsBinaryDeleted(path string) bool {
	// A non-absolute path means we only recovered the bare image name from the
	// process snapshot (the case for protected/system processes we couldn't
	// open, e.g. vmmemWSL) — that's "couldn't read the real path", not a
	// confirmed-deleted binary, so don't raise the deleted-binary warning.
	if path == "" || !filepath.IsAbs(path) {
		return false
	}
	_, err := os.Stat(path)
	return os.IsNotExist(err)
}

// detectWindowsServiceSource returns the Windows service name that owns the PID, if any.
func detectWindowsServiceSource(pid int) string {
	services, err := serviceMapForPIDs()
	if err != nil {
		return ""
	}
	return services[pid]
}
