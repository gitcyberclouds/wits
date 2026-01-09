//go:build darwin

package proc

import (
	"errors"
	"fmt"
	"os/exec"
	"strconv"
	"strings"

	"github.com/pranshuparmar/witr/pkg/model"
)

// ReadExtendedInfo assembles the additional process facts.
// Without /proc, we lean on native utilities (ps, lsof, pgrep, launchctl)
func ReadExtendedInfo(pid int) (model.MemoryInfo, model.IOStats, []string, int, uint64, []int, int, error) {
	memInfo, threadCount, memErr := readDarwinMemory(pid)
	fdCount, fileDescs, fdErr := collectDarwinFDs(pid)
	fdLimit := detectDarwinFileLimit()
	children := listDarwinChildren(pid)

	// macOS only exposes I/O counters via private APIs that require entitlements
	// we do not have. Leave ioStats zeroed.
	// TODO: teach ReadExtendedInfo to use proc_pid_rusage when we can ship the
	// necessary cgo shim without elevated privileges.
	var ioStats model.IOStats

	if memErr != nil && fdErr != nil {
		return memInfo, ioStats, fileDescs, fdCount, fdLimit, children, threadCount, errors.Join(memErr, fdErr)
	}

	return memInfo, ioStats, fileDescs, fdCount, fdLimit, children, threadCount, nil
}

// readDarwinMemory asks ps(1) for RSS, VMS and thread counts.
func readDarwinMemory(pid int) (model.MemoryInfo, int, error) {
	var memInfo model.MemoryInfo
	cmd := exec.Command("ps", "-p", strconv.Itoa(pid), "-o", "rss=,vsz=,thcount=")
	cmd.Env = buildEnvForPS()
	out, err := cmd.Output()
	if err != nil {
		return memInfo, 0, fmt.Errorf("ps rss/vsz: %w", err)
	}
	fields := strings.Fields(strings.TrimSpace(string(out)))
	if len(fields) < 3 {
		return memInfo, 0, fmt.Errorf("ps rss/vsz output missing fields: %q", strings.TrimSpace(string(out)))
	}
	if rss, err := strconv.ParseUint(fields[0], 10, 64); err == nil {
		memInfo.RSS = rss * 1024
		memInfo.RSSMB = float64(memInfo.RSS) / (1024 * 1024)
	}
	if vms, err := strconv.ParseUint(fields[1], 10, 64); err == nil {
		memInfo.VMS = vms * 1024
		memInfo.VMSMB = float64(memInfo.VMS) / (1024 * 1024)
	}
	threadCount, err := strconv.Atoi(fields[2])
	if err != nil {
		threadCount = 0
	}
	return memInfo, threadCount, nil
}

func collectDarwinFDs(pid int) (int, []string, error) {
	cmd := exec.Command("lsof", "-nP", "-p", strconv.Itoa(pid))
	out, err := cmd.Output()
	if err != nil {
		return 0, nil, fmt.Errorf("lsof: %w", err)
	}
	var (
		count   int
		samples []string
		skipHdr = true
	)
	for line := range strings.Lines(string(out)) {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}
		if skipHdr {
			skipHdr = false
			continue
		}
		count++
		if len(samples) < 10 {
			if sample := summarizeLsofLine(trimmed); sample != "" {
				samples = append(samples, sample)
			}
		}
	}
	return count, samples, nil
}

// summarizeLsofLine converts a single lsof(8) row into "FD TYPE TARGET" so
// RenderStandard can display a friendly snippet without dumping dozens of
// columns.
func summarizeLsofLine(line string) string {
	fields := strings.Fields(line)
	if len(fields) < 9 {
		return ""
	}
	fd := fields[3]
	typ := fields[4]
	name := strings.Join(fields[8:], " ")
	return fmt.Sprintf("%s %-4s %s", fd, typ, name)
}

// detectDarwinFileLimit reads launchctl's maxfiles limit (soft cap) so we can
// compute descriptor headroom, falling back to the shell's ulimit if launchctl
// is unavailable.
func detectDarwinFileLimit() uint64 {
	if data, err := exec.Command("launchctl", "limit", "maxfiles").Output(); err == nil {
		for line := range strings.Lines(string(data)) {
			if strings.Contains(line, "maxfiles") {
				if limit, ok := parseLaunchctlLimitLine(line); ok {
					return limit
				}
			}
		}
	}
	if data, err := exec.Command("sh", "-c", "ulimit -n").Output(); err == nil {
		if limit, err := strconv.ParseUint(strings.TrimSpace(string(data)), 10, 64); err == nil {
			return limit
		}
	}
	return 0
}

func parseLaunchctlLimitLine(line string) (uint64, bool) {
	fields := strings.Fields(line)
	if len(fields) < 2 {
		return 0, false
	}
	soft := fields[1]
	if strings.EqualFold(soft, "unlimited") {
		return 0, true
	}
	limit, err := strconv.ParseUint(soft, 10, 64)
	if err != nil {
		return 0, false
	}
	return limit, true
}

// Wrapper around pgrep(1).
func listDarwinChildren(pid int) []int {
	cmd := exec.Command("pgrep", "-P", strconv.Itoa(pid))
	out, err := cmd.CombinedOutput()
	if err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) && exitErr.ExitCode() == 1 {
			return nil
		}
		return nil
	}
	var children []int
	for line := range strings.Lines(string(out)) {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}
		if pidVal, err := strconv.Atoi(trimmed); err == nil {
			children = append(children, pidVal)
		}
	}
	return children
}
