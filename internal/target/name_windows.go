//go:build windows

package target

import (
	"fmt"
	"os"
	"strings"

	procpkg "github.com/pranshuparmar/witr/internal/proc"
)

// ResolveName matches running processes by name (and, when needed, command
// line). The first pass uses ToolHelp32 (instant, no PowerShell) and matches
// against the executable basename. If nothing matches and the user wants
// fuzzy/exact-token matching that includes the command line, only then do we
// pay for per-PID PEB reads — bounded to candidates that survive a cheap
// pre-filter.
func ResolveName(name string, exact bool) ([]int, error) {
	procs, err := procpkg.ListProcessSnapshot()
	if err != nil {
		return nil, fmt.Errorf("enumerate processes: %w", err)
	}

	lowerName := strings.ToLower(name)
	selfPid := os.Getpid()
	ignoredPids := map[int]bool{selfPid: true}
	if ancestry, err := procpkg.ResolveAncestry(selfPid); err == nil {
		for _, p := range ancestry {
			ignoredPids[p.PID] = true
		}
	}

	// Pass 1: name-only match against the executable basename. This is
	// equivalent to what `Get-CimInstance Win32_Process | Select Name`
	// returned and covers the typical `witr chrome.exe` case.
	var pids []int
	var nameMatched bool
	for _, p := range procs {
		if ignoredPids[p.PID] {
			continue
		}
		exeLower := strings.ToLower(p.Command)
		var match bool
		if exact {
			match = exeLower == lowerName
		} else {
			match = strings.Contains(exeLower, lowerName)
		}
		if match {
			pids = append(pids, p.PID)
			nameMatched = true
		}
	}
	if nameMatched {
		return pids, nil
	}

	// Pass 2: cmdline match. Resolving the command line on Windows requires
	// reading each candidate's PEB, which is more expensive than the snapshot
	// scan. Only do it when the name pass produced nothing, and bound the
	// work by skipping ignored PIDs.
	for _, p := range procs {
		if ignoredPids[p.PID] {
			continue
		}
		// Best-effort PEB read; if denied (e.g. SYSTEM-owned), skip.
		info, err := procpkg.GetProcessDetailedInfo(p.PID)
		if err != nil {
			continue
		}
		cmdLower := strings.ToLower(info.CommandLine)
		var match bool
		if exact {
			match = matchesExactToken(cmdLower, lowerName)
		} else {
			match = strings.Contains(cmdLower, lowerName)
		}
		if match {
			pids = append(pids, p.PID)
		}
	}

	if len(pids) == 0 {
		return nil, fmt.Errorf("no process found matching: %s", name)
	}
	return pids, nil
}
