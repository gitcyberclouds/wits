//go:build windows

package proc

import (
	"fmt"
	"os"
	"os/exec"
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

	ports, addrs := GetListeningPortsForPID(pid)
	serviceName := detectWindowsServiceSource(pid)
	container := detectContainer(info.CommandLine)

	return model.Process{
		PID:            pid,
		PPID:           info.PPID,
		Command:        name,
		Cmdline:        info.CommandLine,
		Exe:            info.Exe,
		StartedAt:      info.StartedAt,
		User:           readUser(pid),
		WorkingDir:     info.Cwd,
		ListeningPorts: ports,
		BindAddresses:  addrs,
		Health:         "healthy",
		Forked:         "unknown",
		Env:            info.Env,
		Service:        serviceName,
		Container:      container,
		ExeDeleted:     isWindowsBinaryDeleted(info.Exe),
	}, nil
}

func isWindowsBinaryDeleted(path string) bool {
	if path == "" {
		return false
	}
	_, err := os.Stat(path)
	return os.IsNotExist(err)
}

// detectWindowsServiceSource checks if a PID belongs to a Windows Service via Get-CimInstance.
// Keeping this as a fallback/auxiliary check for now.
func detectWindowsServiceSource(pid int) string {
	psScript := fmt.Sprintf("Get-CimInstance -ClassName Win32_Service -Filter \"ProcessId=%d\" | Select-Object -ExpandProperty Name", pid)
	cmd := exec.Command("powershell", "-NoProfile", "-NonInteractive", psScript)
	out, err := cmd.Output()
	if err != nil {
		return ""
	}

	return strings.TrimSpace(string(out))
}

func detectContainer(cmdline string) string {
	if cmdline == "" {
		return ""
	}
	lowerCmd := strings.ToLower(cmdline)

	switch {
	case strings.Contains(lowerCmd, "docker"):
		if name := extractFlagValue(cmdline, "--name"); name != "" {
			return "docker: " + name
		}
		return "docker"
	case strings.Contains(lowerCmd, "podman"):
		if name := extractFlagValue(cmdline, "--name"); name != "" {
			return "podman: " + name
		}
		return "podman"
	case strings.Contains(lowerCmd, "minikube"):
		if profile := extractFlagValue(cmdline, "-p", "--profile"); profile != "" {
			return "k8s: " + profile
		}
		return "kubernetes"
	case strings.Contains(lowerCmd, "kind"):
		if name := extractFlagValue(cmdline, "--name"); name != "" {
			return "k8s: " + name
		}
		return "kubernetes"
	case strings.Contains(lowerCmd, "kubepods"):
		if id := findLongHexID(cmdline); id != "" {
			if name := resolveContainerName(id, "crictl"); name != "" {
				return "k8s: " + name
			}
			return "k8s (" + id[:12] + ")"
		}
		return "kubernetes"
	case strings.Contains(lowerCmd, "nerdctl"):
		if name := extractFlagValue(cmdline, "--name"); name != "" {
			return "containerd: " + name
		}
		return "containerd"
	case strings.Contains(lowerCmd, "containerd"):
		if name := extractFlagValue(cmdline, "--name"); name != "" {
			return "containerd: " + name
		}
		return "containerd"
	}

	return ""
}
