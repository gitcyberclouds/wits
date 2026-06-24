//go:build linux

package proc

import (
	"bytes"
	"fmt"
	"os/exec"
	"strings"
)

// ResolveSystemdService attempts to find the systemd service name associated with a port.
// It uses `systemctl list-sockets` to find the socket unit and then maps it to the service unit.
func ResolveSystemdService(port int) (string, error) {
	// check if systemctl is available
	if _, err := exec.LookPath("systemctl"); err != nil {
		return "", fmt.Errorf("systemctl not found")
	}

	cmd := exec.Command("systemctl", "list-sockets", "--no-legend", "--full")
	var out bytes.Buffer
	cmd.Stdout = &out
	if err := cmd.Run(); err != nil {
		return "", err
	}

	portStr := fmt.Sprintf(":%d", port)
	lines := strings.Split(out.String(), "\n")

	for _, line := range lines {
		fields := strings.Fields(line)
		if len(fields) < 3 {
			continue
		}
		if strings.HasSuffix(fields[0], portStr) {
			return fields[2], nil
		}
	}

	return "", fmt.Errorf("no systemd service found for port %d", port)
}
