//go:build freebsd

package proc

import (
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/pranshuparmar/witr/pkg/model"
)

func ReadProcess(pid int) (model.Process, error) {
	// Reject PID 0 (and negatives): on FreeBSD `ps -p 0` returns the kernel
	// swapper, which is not a real userland target. Matches the other platforms.
	if pid <= 0 {
		return model.Process{}, fmt.Errorf("invalid pid %d", pid)
	}
	pidStr := strconv.Itoa(pid)

	// Format: pid(0) ppid(1) uid(2) jid(3) state(4) pcpu(5) rss(6) lstart(7-11) args(12+)
	// comm is excluded because it can contain spaces, which breaks strings.Fields parsing.
	// The display name is derived from args instead.
	cmd := exec.Command("ps", "-p", pidStr,
		"-o", "pid=", "-o", "ppid=", "-o", "uid=", "-o", "jid=",
		"-o", "state=", "-o", "pcpu=", "-o", "rss=",
		"-o", "lstart=", "-o", "args=")
	cmd.Env = buildEnvForPS()
	out, err := cmd.Output()
	if err != nil {
		return model.Process{}, fmt.Errorf("process %d not found: %w", pid, err)
	}

	line := strings.TrimSpace(string(out))
	if line == "" {
		return model.Process{}, fmt.Errorf("process %d not found", pid)
	}

	fields := strings.Fields(line)
	if len(fields) < 12 {
		return model.Process{}, fmt.Errorf("unexpected ps output format for pid %d: got %d fields in %q", pid, len(fields), line)
	}

	ppid, _ := strconv.Atoi(fields[1])
	uid, _ := strconv.Atoi(fields[2])
	jid := fields[3]
	state := fields[4]
	cpuPct, _ := strconv.ParseFloat(fields[5], 64)
	rssKB, _ := strconv.ParseFloat(fields[6], 64)

	lstartStr := strings.Join(fields[7:12], " ")
	startedAt := parseLstart(lstartStr)
	if startedAt.IsZero() {
		startedAt = time.Now().UTC()
	}

	rawCmdline := ""
	if len(fields) > 12 {
		rawCmdline = strings.Join(fields[12:], " ")
	}
	cmdline := rawCmdline

	cwd, binPath := getCwdAndBinaryPath(pid)
	env := getEnvironment(pid)

	health := "healthy"
	var forked string

	// FreeBSD states can be multi-character like "Is", "Ss", "R", "Z", "T"
	if len(state) > 0 {
		switch state[0] {
		case 'Z':
			health = "zombie"
		case 'T':
			health = "stopped"
		}
	}

	if health == "healthy" && cpuPct > 90 {
		health = "high-cpu"
	}
	rssMB := rssKB / 1024
	if health == "healthy" && rssMB > 1024 {
		health = "high-mem"
	}

	memBytes := uint64(rssKB * 1024)
	memPercent := 0.0
	if total := totalMemoryBytes(); total > 0 {
		memPercent = float64(memBytes) / float64(total) * 100.0
	}

	// Display name resolution order:
	//   1. filepath.Base(binPath) — binPath comes from `procstat -f` as a
	//      single unsplit line, preserving any spaces in the path.
	//   2. `ps -p <pid> -o comm=` on its own line — also preserves spaces.
	//   3. extractExecutableName(rawCmdline) — last resort. `ps -o args=`
	//      joins argv with single spaces and does not quote paths, so this
	//      can truncate names when the executable path contains spaces
	//      (issue #201).
	displayName := binaryBasename(binPath)
	if displayName == "" {
		if commOut, commErr := exec.Command("ps", "-p", pidStr, "-o", "comm=").Output(); commErr == nil {
			displayName = binaryBasename(string(commOut))
		}
	}
	if displayName == "" {
		displayName = extractExecutableName(rawCmdline)
	}
	if cmdline == "" {
		cmdline = displayName
	}

	if ppid != 1 && displayName != "init" {
		forked = "forked"
	} else {
		forked = "not-forked"
	}

	user := readUserByUID(uid)
	container := detectContainerFreeBSD(jid, cmdline)

	if displayName == "docker-proxy" && container == "" {
		container = resolveDockerProxyContainer(cmdline)
	}

	service := detectRcService(pid)
	gitRepo, gitBranch := detectGitInfo(cwd)
	procSockets := socketsForPID(pid)

	exeDeleted := false
	if binPath != "" {
		_, statErr := os.Stat(binPath)
		exeDeleted = os.IsNotExist(statErr)
	}

	return model.Process{
		PID:           pid,
		PPID:          ppid,
		Command:       displayName,
		Cmdline:       cmdline,
		StartedAt:     startedAt,
		User:          user,
		CPUPercent:    cpuPct,
		MemoryRSS:     memBytes,
		MemoryPercent: memPercent,
		WorkingDir:    cwd,
		GitRepo:       gitRepo,
		GitBranch:     gitBranch,
		Container:     container,
		Service:       service,
		Sockets:       procSockets,
		Health:        health,
		Forked:        forked,
		Env:           env,
		ExeDeleted:    exeDeleted,
	}, nil
}

var (
	totalMemOnce  sync.Once
	totalMemBytes uint64
)

// totalMemoryBytes returns total physical RAM in bytes via sysctl hw.physmem,
// or 0 if it can't be read. The value is constant for the machine, so it is
// resolved once rather than spawning sysctl on every ancestry hop.
func totalMemoryBytes() uint64 {
	totalMemOnce.Do(func() {
		out, err := exec.Command("sysctl", "-n", "hw.physmem").Output()
		if err != nil {
			return
		}
		totalMemBytes, _ = strconv.ParseUint(strings.TrimSpace(string(out)), 10, 64)
	})
	return totalMemBytes
}

// getCwdAndBinaryPath returns the working directory and executable path for a process.
func getCwdAndBinaryPath(pid int) (cwd string, binPath string) {
	cwd = "unknown"

	out, err := exec.Command("procstat", "-f", strconv.Itoa(pid)).Output()
	if err != nil {
		return cwd, ""
	}

	// procstat -f output format: PID COMM FD TYPE FLAGS ... PATH
	for line := range strings.Lines(string(out)) {
		fields := strings.Fields(line)
		if len(fields) < 4 {
			continue
		}
		switch fields[2] {
		case "cwd":
			cwd = fields[len(fields)-1]
		case "text":
			if binPath == "" {
				binPath = fields[len(fields)-1]
			}
		}
	}

	return cwd, binPath
}

func parseLstart(lstartStr string) time.Time {
	if lstartStr == "" {
		return time.Time{}
	}
	// FreeBSD lstart format with LC_ALL=C: "Thu Jan  2 10:26:00 2025"
	// strings.Fields collapses double spaces, so try the standard format.
	formats := []string{
		"Mon Jan 2 15:04:05 2006",
		"Mon Jan  2 15:04:05 2006",
		"Mon Jan 02 15:04:05 2006",
	}
	for _, format := range formats {
		if t, err := time.Parse(format, lstartStr); err == nil {
			return t
		}
	}
	return time.Time{}
}

func getEnvironment(pid int) []string {
	var env []string

	// Use procstat -e to get environment variables
	// procstat does not require procfs to be mounted
	out, err := exec.Command("procstat", "-e", strconv.Itoa(pid)).Output()
	if err != nil {
		return env
	}

	// Parse procstat -e output
	// Format: PID COMM ENVVAR=VALUE ...
	for line := range strings.Lines(string(out)) {
		fields := strings.Fields(line)
		if len(fields) < 3 {
			continue
		}
		// Skip header and PID/COMM columns
		for _, field := range fields[2:] {
			if strings.Contains(field, "=") {
				env = append(env, field)
			}
		}
	}

	return env
}

// detectContainerFreeBSD checks for jail membership first, then falls back
// to cmdline-based container detection shared with other platforms.
func detectContainerFreeBSD(jid, cmdline string) string {
	if jid != "" && jid != "0" {
		if name := resolveJailName(jid); name != "" {
			return "jail: " + name
		}
		return "jail (" + jid + ")"
	}
	return detectContainerFromCmdline(cmdline)
}

func detectRcService(pid int) string {
	// FreeBSD uses rc.d for service management
	// Try to find the service by checking /var/run/*.pid files
	pidStr := strconv.Itoa(pid)

	entries, err := os.ReadDir("/var/run")
	if err != nil {
		return ""
	}

	for _, entry := range entries {
		if !strings.HasSuffix(entry.Name(), ".pid") {
			continue
		}

		pidFile := "/var/run/" + entry.Name()
		content, err := os.ReadFile(pidFile)
		if err != nil {
			continue
		}

		if strings.TrimSpace(string(content)) == pidStr {
			// Found matching PID file
			serviceName := strings.TrimSuffix(entry.Name(), ".pid")
			return serviceName
		}
	}

	return ""
}

func resolveJailName(jid string) string {
	out, err := exec.Command("jls", "-j", jid, "name").Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}
