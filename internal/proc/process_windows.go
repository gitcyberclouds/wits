//go:build windows

package proc

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/pranshuparmar/witr/pkg/model"
)

func ReadProcess(pid int) (model.Process, error) {
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

	return model.Process{
		PID:        pid,
		PPID:       info.PPID,
		Command:    name,
		Cmdline:    info.CommandLine,
		Exe:        info.Exe,
		StartedAt:  info.StartedAt,
		User:       readUser(pid),
		WorkingDir: info.Cwd,
		GitRepo:    gitRepo,
		GitBranch:  gitBranch,
		Sockets:    procSockets,
		Health:     "healthy",
		Forked:     "unknown",
		Env:        info.Env,
		Service:    serviceName,
		Container:  container,
		ExeDeleted: isWindowsBinaryDeleted(info.Exe),
	}, nil
}

func isWindowsBinaryDeleted(path string) bool {
	if path == "" {
		return false
	}
	_, err := os.Stat(path)
	return os.IsNotExist(err)
}

// detectWindowsServiceSource checks if a PID belongs to a Windows Service via
// Get-CimInstance. Bounded by a short timeout so a stalled WMI provider can't
// hang the report — on timeout we return an empty string and the Service line
// is just omitted.
func detectWindowsServiceSource(pid int) string {
	psScript := fmt.Sprintf("Get-CimInstance -ClassName Win32_Service -Filter \"ProcessId=%d\" | Select-Object -ExpandProperty Name", pid)
	out, err := runPowerShell(psScript)
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}
