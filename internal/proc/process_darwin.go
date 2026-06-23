//go:build darwin

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
	if pid <= 0 {
		return model.Process{}, fmt.Errorf("invalid pid %d", pid)
	}
	pidStr := strconv.Itoa(pid)

	// Format: pid(0) ppid(1) uid(2) lstart(3-7) state(8) pcpu(9) rss(10) args(11+)
	// ucomm is excluded because it can contain spaces (e.g. "Microsoft Teams"),
	// which breaks strings.Fields parsing. The display name is derived from args instead.
	cmd := exec.Command("ps", "-p", pidStr, "-o", "pid=,ppid=,uid=,lstart=,state=,pcpu=,rss=,args=")
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
	if len(fields) < 11 {
		return model.Process{}, fmt.Errorf("unexpected ps output format for pid %d", pid)
	}

	ppid, _ := strconv.Atoi(fields[1])
	uid, _ := strconv.Atoi(fields[2])

	lstartStr := strings.Join(fields[3:8], " ")
	startedAt, _ := time.Parse("Mon Jan 2 15:04:05 2006", lstartStr)
	if startedAt.IsZero() {
		startedAt = time.Now().UTC()
	}

	state := fields[8]

	cpuPct, _ := strconv.ParseFloat(fields[9], 64)
	rssKB, _ := strconv.ParseFloat(fields[10], 64)

	rawCmdline := ""
	if len(fields) > 11 {
		rawCmdline = strings.Join(fields[11:], " ")
	}
	cmdline := rawCmdline

	env := getEnvironment(pid)
	cwd, binPath := getCwdAndBinaryPath(pid)

	health := "healthy"
	var forked string

	switch state {
	case "Z":
		health = "zombie"
	case "T":
		health = "stopped"
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
	//   1. filepath.Base(binPath) — binPath comes from `lsof ftxt` as a single
	//      unsplit line, so spaces in .app bundle paths are preserved.
	//   2. `ps -p <pid> -o comm=` on its own line — also preserves spaces.
	//   3. extractExecutableName(rawCmdline) — last resort. `ps -o args=` joins
	//      argv with single spaces and does not quote paths, so this can
	//      truncate names when the executable path contains spaces (issue #201).
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

	if ppid != 1 && displayName != "launchd" {
		forked = "forked"
	} else {
		forked = "not-forked"
	}

	user := readUserByUID(uid)
	container := detectContainerFromCmdline(cmdline)

	if displayName == "docker-proxy" && container == "" {
		container = resolveDockerProxyContainer(cmdline)
	}

	service := detectLaunchdService(pid)
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

// totalMemoryBytes returns total physical RAM in bytes via sysctl hw.memsize,
// or 0 if it can't be read. The value is constant for the machine, so it is
// resolved once rather than spawning sysctl on every ancestry hop.
func totalMemoryBytes() uint64 {
	totalMemOnce.Do(func() {
		out, err := exec.Command("sysctl", "-n", "hw.memsize").Output()
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

	// lsof may exit non-zero when one of the requested FDs (e.g., txt for a
	// deleted/inaccessible binary) is unavailable, but still emit valid cwd
	// data on stdout. Salvage stdout when present instead of bailing out.
	out, err := exec.Command("lsof", "-a", "-p", strconv.Itoa(pid), "-d", "cwd,txt", "-F", "fn").Output()
	if err != nil && len(out) == 0 {
		return cwd, ""
	}

	// lsof -F fn output has lines like:
	//   p<pid>
	//   fcwd
	//   n/path/to/cwd
	//   ftxt
	//   n/path/to/binary
	currentFD := ""
	for line := range strings.Lines(string(out)) {
		if len(line) < 2 {
			continue
		}
		switch line[0] {
		case 'f':
			currentFD = strings.TrimSpace(line[1:])
		case 'n':
			path := strings.TrimSpace(line[1:])
			switch currentFD {
			case "cwd":
				cwd = path
			case "txt":
				if binPath == "" {
					binPath = path
				}
			}
		}
	}

	return cwd, binPath
}

func getEnvironment(pid int) []string {
	var env []string

	// On macOS, getting environment of another process requires elevated privileges
	// or using the proc_pidinfo syscall. For simplicity, we use ps -E when available
	// Note: This might not work for all processes due to SIP restrictions

	out, err := exec.Command("ps", "-p", strconv.Itoa(pid), "-E", "-o", "command=").Output()
	if err != nil {
		return env
	}

	// The -E output appends environment to the command
	// This is a simplified approach; full env parsing would need libproc
	output := string(out)

	// Look for common environment variable patterns
	for _, part := range strings.Fields(output) {
		if strings.Contains(part, "=") && !strings.HasPrefix(part, "-") {
			// Basic validation - should look like VAR=value
			eqIdx := strings.Index(part, "=")
			if eqIdx > 0 {
				varName := part[:eqIdx]
				// Check if it looks like an env var name (uppercase or common patterns)
				if isEnvVarName(varName) {
					env = append(env, part)
				}
			}
		}
	}

	return env
}

func isEnvVarName(name string) bool {
	if len(name) == 0 {
		return false
	}
	// Common env var patterns
	for _, c := range name {
		if !((c >= 'A' && c <= 'Z') || (c >= 'a' && c <= 'z') || (c >= '0' && c <= '9') || c == '_') {
			return false
		}
	}
	return true
}

func detectLaunchdService(pid int) string {
	// Try to find the launchd service managing this process
	// Use launchctl blame on macOS 10.10+

	out, err := exec.Command("launchctl", "blame", strconv.Itoa(pid)).Output()
	if err == nil {
		blame := strings.TrimSpace(string(out))
		if blame != "" && !strings.Contains(blame, "unknown") {
			return blame
		}
	}

	// Fallback: check if process is a known launchd service
	// by looking at the parent chain or service database
	return ""
}
