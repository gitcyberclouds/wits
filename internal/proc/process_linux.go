//go:build linux

package proc

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/pranshuparmar/witr/pkg/model"
)

func ReadProcess(pid int) (model.Process, error) {
	if pid <= 0 {
		return model.Process{}, fmt.Errorf("invalid pid %d", pid)
	}
	// Verify process still exists before reading
	if _, err := os.Stat(fmt.Sprintf("/proc/%d", pid)); os.IsNotExist(err) {
		return model.Process{}, fmt.Errorf("process %d does not exist", pid)
	}

	// Read all proc files in a logical order to minimize TOCTOU issues
	// Start with stat file which is most likely to fail if process disappears
	statPath := fmt.Sprintf("/proc/%d/stat", pid)
	stat, err := os.ReadFile(statPath)
	if err != nil {
		return model.Process{}, fmt.Errorf("process %d disappeared during read", pid)
	}

	// Read environment variables
	env := []string{}
	envBytes, errEnv := os.ReadFile(fmt.Sprintf("/proc/%d/environ", pid))
	if errEnv == nil {
		for _, e := range strings.Split(string(envBytes), "\x00") {
			if e != "" {
				env = append(env, e)
			}
		}
	}
	// Health status
	health := "healthy"

	// Working directory
	var cwd, cwdErr = os.Readlink(fmt.Sprintf("/proc/%d/cwd", pid))
	if cwdErr != nil {
		cwd = "unknown"
	} else if cwd == "" {
		cwd = "invalid"
	}

	// Container detection
	container := ""
	var containerID, containerRuntime string
	cgroupFile := fmt.Sprintf("/proc/%d/cgroup", pid)
	if cgroupData, err := os.ReadFile(cgroupFile); err == nil {
		cgroupStr := string(cgroupData)
		switch {
		case strings.Contains(cgroupStr, "docker"):
			container = "docker"
			containerRuntime = "docker"
			containerID = extractContainerID(cgroupStr, "docker-", "docker/")
			if containerID != "" {
				if name := resolveContainerName(containerID, "docker"); name != "" {
					container = name
				} else {
					container = "docker (" + shortID(containerID) + ")"
				}
			}

		case strings.Contains(cgroupStr, "podman"), strings.Contains(cgroupStr, "libpod"):
			container = "podman"
			containerRuntime = "podman"
			containerID = extractContainerID(cgroupStr, "libpod-", "libpod/")
			if containerID != "" {
				if name := resolveContainerName(containerID, "podman"); name != "" {
					container = name
				} else {
					container = "podman (" + shortID(containerID) + ")"
				}
			}

		case strings.Contains(cgroupStr, "kubepods"):
			container = "kubernetes"
			containerRuntime = "crictl"
			if id := findLongHexID(cgroupStr); id != "" {
				containerID = id
				if name := resolveContainerName(containerID, "crictl"); name != "" {
					container = "k8s: " + name
				} else {
					container = "k8s (" + shortID(containerID) + ")"
				}
			}

		case strings.Contains(cgroupStr, "containerd"):
			container = "containerd"
			containerRuntime = "nerdctl"
			if id := findLongHexID(cgroupStr); id != "" {
				containerID = id
				if name := resolveContainerName(containerID, "nerdctl"); name != "" {
					container = "containerd: " + name
				} else {
					container = "containerd (" + shortID(containerID) + ")"
				}
			}

		case strings.Contains(cgroupStr, "colima"):
			container = "colima"
			if idx := strings.Index(cgroupStr, "colima-"); idx != -1 {
				rest := cgroupStr[idx+7:]
				if dot := strings.Index(rest, ".scope"); dot != -1 {
					container = "colima: " + rest[:dot]
				}
			} else if strings.Contains(cgroupStr, "colima") {
				container = "colima: default"
			}
		case strings.Contains(cgroupStr, "lxc.payload"):
			name := extractLXCBasedContainerName(cgroupStr)
			if name != "" {
				container = "lxc-based: " + name
			} else {
				container = "lxc-based"
			}
		}
	}

	// Snap/Flatpak sandbox detection via environment variables
	if container == "" {
		for _, e := range env {
			if strings.HasPrefix(e, "SNAP_NAME=") {
				container = "snap: " + e[len("SNAP_NAME="):]
				break
			}
			if strings.HasPrefix(e, "FLATPAK_ID=") {
				container = "flatpak: " + e[len("FLATPAK_ID="):]
				break
			}
		}
	}

	// Resolve the owning systemd .service from the process cgroup — a cheap file
	// read with no subprocess. (This replaced a per-ancestor `systemctl status`
	// probe whose parser never matched, so the field used to be empty.)
	service := serviceFromCgroup(pid)

	gitRepo, gitBranch := detectGitInfo(cwd)

	// stat format is evil, command is inside ()
	raw := string(stat)
	open := strings.Index(raw, "(")
	close := strings.LastIndex(raw, ")")
	if open == -1 || close == -1 || close+2 >= len(raw) {
		return model.Process{}, fmt.Errorf("invalid stat format for pid %d", pid)
	}

	comm := raw[open+1 : close]
	fields := strings.Fields(raw[close+2:])
	// /proc/[pid]/stat has 52 fields after comm; we need at least index 21 (rss)
	if len(fields) < 22 {
		return model.Process{}, fmt.Errorf("unexpected stat format for pid %d: got %d fields", pid, len(fields))
	}

	ppid, _ := strconv.Atoi(fields[1])
	state := processState(fields)
	startTicks, _ := strconv.ParseInt(fields[19], 10, 64)

	// Fork detection: if ppid != 1 and not systemd, likely forked; also check for vfork/fork/clone flags if possible
	var forked string
	if ppid != 1 && comm != "systemd" {
		forked = "forked"
	} else {
		forked = "not-forked"
	}

	startedAt := startTimeFromTicks(bootTime(), startTicks, ticksPerSecond())

	// Health: zombie/stopped
	switch state {
	case "Z":
		health = "zombie"
	case "T":
		health = "stopped"
	}

	// Flag high CPU (>2h total) and high memory (>1GB RSS).
	utime, _ := strconv.ParseFloat(fields[11], 64)
	stime, _ := strconv.ParseFloat(fields[12], 64)
	rssPages, _ := strconv.ParseFloat(fields[21], 64)
	clkTck := float64(ticksPerSecond())
	totalCPU := (utime + stime) / clkTck
	if health == "healthy" && totalCPU > 60*60*2 { // >2h CPU time
		health = "high-cpu"
	}
	pageSize := float64(os.Getpagesize())
	memBytes := rssPages * pageSize
	memMB := memBytes / (1024 * 1024)
	if health == "healthy" && memMB > 1024 {
		health = "high-mem"
	}

	memPercent := 0.0
	if total := totalMemoryBytes(); total > 0 {
		memPercent = memBytes / float64(total) * 100.0
	}

	// Lifetime-average CPU%: total CPU time over wall-clock time since start.
	cpuPercent := 0.0
	if wall := time.Since(startedAt).Seconds(); wall > 0 {
		cpuPercent = totalCPU / wall * 100.0
	}

	user := readUser(pid)

	sockets, _ := readSocketsCached()
	inodes := socketsForPID(pid)

	var procSockets []model.Socket

	// Check for IPv4 listeners first to avoid duplicates when synthesizing
	ipv4Listeners := make(map[int]bool)
	for _, inode := range inodes {
		if s, ok := sockets[inode]; ok {
			if s.State != "LISTEN" {
				continue
			}
			if s.Address == "0.0.0.0" {
				ipv4Listeners[s.Port] = true
			}
		}
	}

	dualStackAllowed := isDualStackEnabled()

	for _, inode := range inodes {
		if s, ok := sockets[inode]; ok {
			procSockets = append(procSockets, s)

			// Heuristic: If system allows dual-stack, we see a `::` listener,
			// and there is NO explicit 0.0.0.0 listener, synthesize the implicit
			// IPv4 mapping so users see what the kernel actually accepts.
			if dualStackAllowed && s.State == "LISTEN" && s.Address == "::" && !ipv4Listeners[s.Port] {
				v4 := s
				v4.Address = "0.0.0.0"
				v4.Protocol = strings.TrimSuffix(s.Protocol, "6")
				procSockets = append(procSockets, v4)
			}
		}
	}
	// Full command line
	cmdline := ""
	cmdlineBytes, err := os.ReadFile(fmt.Sprintf("/proc/%d/cmdline", pid))
	if err == nil {
		cmd := strings.ReplaceAll(string(cmdlineBytes), "\x00", " ")
		cmdline = strings.TrimSpace(cmd)
	}

	// Recover full process name when kernel comm field is truncated
	displayName := deriveDisplayCommand(comm, cmdline)
	if displayName == "" {
		displayName = comm
	}

	if comm == "docker-proxy" && container == "" {
		container = resolveDockerProxyContainer(cmdline)
	}

	return model.Process{
		PID:              pid,
		PPID:             ppid,
		Command:          displayName,
		Cmdline:          cmdline,
		StartedAt:        startedAt,
		User:             user,
		CPUPercent:       cpuPercent,
		MemoryRSS:        uint64(memBytes),
		MemoryPercent:    memPercent,
		WorkingDir:       cwd,
		GitRepo:          gitRepo,
		GitBranch:        gitBranch,
		Container:        container,
		ContainerID:      containerID,
		ContainerRuntime: containerRuntime,
		Service:          service,
		Sockets:          procSockets,
		Health:           health,
		Forked:           forked,
		Env:              env,
		ExeDeleted:       isBinaryDeleted(pid),
		Capabilities:     ReadCapabilities(pid),
	}, nil
}

var (
	totalMemOnce  sync.Once
	totalMemBytes uint64
)

// totalMemoryBytes returns total physical RAM in bytes from /proc/meminfo, or 0
// if it can't be read. The value is constant for the machine, so it is resolved
// once rather than re-read on every ancestry hop.
func totalMemoryBytes() uint64 {
	totalMemOnce.Do(func() {
		data, err := os.ReadFile("/proc/meminfo")
		if err != nil {
			return
		}
		for _, line := range strings.Split(string(data), "\n") {
			if strings.HasPrefix(line, "MemTotal:") {
				fields := strings.Fields(line)
				if len(fields) >= 2 {
					kb, _ := strconv.ParseUint(fields[1], 10, 64)
					totalMemBytes = kb * 1024
					return
				}
			}
		}
	})
	return totalMemBytes
}

func isBinaryDeleted(pid int) bool {
	exePath, err := os.Readlink(fmt.Sprintf("/proc/%d/exe", pid))
	if err != nil {
		return false
	}
	return strings.HasSuffix(exePath, " (deleted)")
}

// The kernel emits the state immediately after the command, so fields[0] always carries it.
func processState(fields []string) string {
	if len(fields) == 0 {
		return ""
	}
	state := fields[0]
	if len(state) == 0 {
		return ""
	}
	return state[:1]
}

// isDualStackEnabled checks if /proc/sys/net/ipv6/bindv6only is 0 (or missing),
// which implies that IPv6 sockets can handle IPv4 traffic by default.
func isDualStackEnabled() bool {
	data, err := os.ReadFile("/proc/sys/net/ipv6/bindv6only")
	if err != nil {
		return true
	}
	return strings.TrimSpace(string(data)) == "0"
}

func extractContainerID(cgroup, dashPrefix, slashPrefix string) string {
	// Pattern 1: .../prefix-<id>.scope
	if idx := strings.Index(cgroup, dashPrefix); idx != -1 {
		rest := cgroup[idx+len(dashPrefix):]
		if dot := strings.Index(rest, ".scope"); dot != -1 {
			return rest[:dot]
		}
	}
	// Pattern 2: .../prefix/<id>
	if idx := strings.Index(cgroup, slashPrefix); idx != -1 {
		rest := cgroup[idx+len(slashPrefix):]
		if len(rest) >= 64 {
			return rest[:64]
		}
	}
	return ""
}

func extractLXCBasedContainerName(cgroup string) string {
	idx := strings.Index(cgroup, "lxc.payload.")
	if idx == -1 {
		return ""
	}
	rest := cgroup[idx+len("lxc.payload."):]

	// Only strip "user-<uid>_" prefix, not arbitrary underscores
	if strings.HasPrefix(rest, "user-") {
		if u := strings.Index(rest, "_"); u != -1 {
			rest = rest[u+1:]
		}
	}

	if slash := strings.Index(rest, "/"); slash != -1 {
		rest = rest[:slash]
	}
	return rest
}
