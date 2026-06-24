//go:build windows

package target

import (
	"fmt"
	"os/exec"
	"strconv"
	"strings"
)

func ResolvePort(port int) ([]int, error) {
	// netstat -ano
	out, err := exec.Command("netstat", "-ano").Output()
	if err != nil {
		return nil, err
	}

	lines := strings.Split(string(out), "\n")
	portStr := fmt.Sprintf(":%d", port)
	var pids, fallbackPIDs []int
	seen := make(map[int]bool)
	fallbackSeen := make(map[int]bool)
	sawListenNoOwner := false

	for _, line := range lines {
		if !strings.Contains(line, portStr) {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) < 4 {
			continue
		}

		proto := strings.ToUpper(fields[0])
		localAddr := fields[1]
		foreignAddr := fields[2]

		matchesLocal := strings.HasSuffix(localAddr, portStr)
		matchesForeign := strings.HasSuffix(foreignAddr, portStr)
		if !matchesLocal && !matchesForeign {
			continue
		}

		if strings.HasPrefix(proto, "TCP") {
			// TCP: Proto LocalAddr ForeignAddr State PID
			if len(fields) < 5 {
				continue
			}
			pid, _ := strconv.Atoi(fields[4])
			isListen := matchesLocal && fields[3] == "LISTENING"
			if pid == 0 {
				if isListen {
					sawListenNoOwner = true
				}
				continue
			}
			if isListen {
				if !seen[pid] {
					pids = append(pids, pid)
					seen[pid] = true
				}
			} else if !fallbackSeen[pid] {
				fallbackPIDs = append(fallbackPIDs, pid)
				fallbackSeen[pid] = true
			}
		} else if strings.HasPrefix(proto, "UDP") {
			// UDP: Proto LocalAddr *:* PID (no state column)
			if !matchesLocal || len(fields) < 4 {
				continue
			}
			pid, _ := strconv.Atoi(fields[3])
			if pid != 0 && !seen[pid] {
				pids = append(pids, pid)
				seen[pid] = true
			}
		}
	}

	if len(pids) > 0 {
		return pids, nil
	}
	if len(fallbackPIDs) > 0 {
		return fallbackPIDs, nil
	}
	if sawListenNoOwner {
		return nil, ErrSocketOwnerUnknown
	}
	return nil, fmt.Errorf("no process found listening on port %d", port)
}
