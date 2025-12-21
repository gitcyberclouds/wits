package target

import (
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
)

func ResolveName(name string) ([]int, error) {
	var procPIDs []int

	// Process name and command line matching (case-insensitive, substring)
	entries, _ := os.ReadDir("/proc")
	lowerName := strings.ToLower(name)
	for _, e := range entries {
		pid, err := strconv.Atoi(e.Name())
		if err != nil {
			continue
		}

		comm, err := os.ReadFile("/proc/" + e.Name() + "/comm")
		if err == nil {
			if strings.Contains(strings.ToLower(strings.TrimSpace(string(comm))), lowerName) {
				procPIDs = append(procPIDs, pid)
				continue
			}
		}

		cmdline, err := os.ReadFile("/proc/" + e.Name() + "/cmdline")
		if err == nil {
			// cmdline is null-separated
			cmd := strings.ReplaceAll(string(cmdline), "\x00", " ")
			if strings.Contains(strings.ToLower(cmd), lowerName) {
				procPIDs = append(procPIDs, pid)
			}
		}
	}

	// Service detection (systemd)
	servicePID, serviceErr := resolveSystemdServiceMainPID(name)

	// Ambiguity: both process and service
	if len(procPIDs) > 0 && servicePID > 0 {
		fmt.Printf("Ambiguous target: \"%s\"\n\n", name)
		fmt.Println("The name matches multiple entities:")
		fmt.Println()
		// Service entry first
		fmt.Printf("[1] PID %d   %s: master process   (service)\n", servicePID, name)
		// Process entries (skip if PID matches servicePID)
		idx := 2
		for _, pid := range procPIDs {
			if pid == servicePID {
				continue
			}
			fmt.Printf("[%d] PID %d   %s: worker process   (manual)\n", idx, pid, name)
			idx++
		}
		fmt.Println()
		fmt.Println("witr cannot determine intent safely.")
		fmt.Println("Please re-run with an explicit PID:")
		fmt.Println("  witr --pid <pid>")
		os.Exit(1)
	}

	// Service only
	if servicePID > 0 {
		return []int{servicePID}, nil
	}

	// Process only
	if len(procPIDs) > 0 {
		return procPIDs, nil
	}

	// Neither found
	if serviceErr != nil {
		return nil, fmt.Errorf("no running process or service named %q", name)
	}
	return nil, fmt.Errorf("no running process or service named %q", name)
}

// resolveSystemdServiceMainPID tries to resolve a systemd service and returns its MainPID if running.
func resolveSystemdServiceMainPID(name string) (int, error) {
	// Accept both foo and foo.service
	svcName := name
	if !strings.HasSuffix(svcName, ".service") {
		svcName += ".service"
	}
	out, err := exec.Command("systemctl", "show", svcName, "-p", "MainPID", "--value").Output()
	if err != nil {
		return 0, err
	}
	pidStr := strings.TrimSpace(string(out))
	pid, err := strconv.Atoi(pidStr)
	if err != nil || pid == 0 {
		return 0, fmt.Errorf("service %q not running", svcName)
	}
	return pid, nil
}
