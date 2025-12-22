package proc

import (
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"github.com/pranshuparmar/witr/pkg/model"
)

func ReadProcess(pid int) (model.Process, error) {
	// Working directory
	var cwd, err = os.Readlink(fmt.Sprintf("/proc/%d/cwd", pid))
	if err != nil {
		cwd = "unknown"
	}

	// Container detection (simple: look for docker/containerd/kubepods in cgroup)
	container := ""
	cgroupFile := fmt.Sprintf("/proc/%d/cgroup", pid)
	if cgroupData, err := os.ReadFile(cgroupFile); err == nil {
		cgroupStr := string(cgroupData)
		if strings.Contains(cgroupStr, "docker") {
			container = "docker"
		} else if strings.Contains(cgroupStr, "containerd") {
			container = "containerd"
		} else if strings.Contains(cgroupStr, "kubepods") {
			container = "kubernetes"
		}
	}

	// Service detection (try systemctl show for this PID)
	service := ""
	svcOut, err := exec.Command("systemctl", "status", fmt.Sprintf("%d", pid)).CombinedOutput()
	if err == nil && strings.Contains(string(svcOut), "Loaded: loaded") {
		// Try to extract service name from output
		lines := strings.Split(string(svcOut), "\n")
		for _, line := range lines {
			if strings.HasPrefix(line, "Loaded:") && strings.Contains(line, ".service") {
				parts := strings.Fields(line)
				for _, part := range parts {
					if strings.HasSuffix(part, ".service") {
						service = part
						break
					}
				}
			}
		}
	}

	// Git repo/branch detection (walk up to find .git)
	gitRepo := ""
	gitBranch := ""
	if cwd != "unknown" {
		searchDir := cwd
		for searchDir != "/" && searchDir != "." && searchDir != "" {
			gitDir := searchDir + "/.git"
			if fi, err := os.Stat(gitDir); err == nil && fi.IsDir() {
				// Repo name is the base dir
				parts := strings.Split(strings.TrimRight(searchDir, "/"), "/")
				gitRepo = parts[len(parts)-1]
				// Try to read HEAD for branch
				headFile := gitDir + "/HEAD"
				if head, err := os.ReadFile(headFile); err == nil {
					headStr := strings.TrimSpace(string(head))
					if strings.HasPrefix(headStr, "ref: ") {
						ref := strings.TrimPrefix(headStr, "ref: ")
						refParts := strings.Split(ref, "/")
						gitBranch = refParts[len(refParts)-1]
					}
				}
				break
			}
			// Move up one directory
			idx := strings.LastIndex(searchDir, "/")
			if idx <= 0 {
				break
			}
			searchDir = searchDir[:idx]
		}
	}
	statPath := fmt.Sprintf("/proc/%d/stat", pid)
	stat, err := os.ReadFile(statPath)
	if err != nil {
		return model.Process{}, err
	}

	// stat format is evil, command is inside ()
	raw := string(stat)
	open := strings.Index(raw, "(")
	close := strings.LastIndex(raw, ")")
	if open == -1 || close == -1 {
		return model.Process{}, fmt.Errorf("invalid stat format")
	}

	comm := raw[open+1 : close]
	fields := strings.Fields(raw[close+2:])

	ppid, _ := strconv.Atoi(fields[1])
	startTicks, _ := strconv.ParseInt(fields[19], 10, 64)

	startedAt := bootTime().Add(time.Duration(startTicks) * time.Second / ticksPerSecond())

	user := readUser(pid)

	sockets, _ := readListeningSockets()
	inodes := socketsForPID(pid)

	var ports []int
	var addrs []string

	for _, inode := range inodes {
		if s, ok := sockets[inode]; ok {
			ports = append(ports, s.Port)
			addrs = append(addrs, s.Address)
		}
	}
	// Full command line
	cmdline := ""
	cmdlineBytes, err := os.ReadFile(fmt.Sprintf("/proc/%d/cmdline", pid))
	if err == nil {
		cmd := strings.ReplaceAll(string(cmdlineBytes), "\x00", " ")
		cmdline = strings.TrimSpace(cmd)
	}

	return model.Process{
		PID:            pid,
		PPID:           ppid,
		Command:        comm,
		Cmdline:        cmdline,
		StartedAt:      startedAt,
		User:           user,
		WorkingDir:     cwd,
		GitRepo:        gitRepo,
		GitBranch:      gitBranch,
		Container:      container,
		Service:        service,
		ListeningPorts: ports,
		BindAddresses:  addrs,
	}, nil
}
