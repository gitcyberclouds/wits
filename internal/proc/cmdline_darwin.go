//go:build darwin

package proc

import (
	"os/exec"
	"strconv"
	"strings"
)

// GetCmdline returns the command line for a given PID
func GetCmdline(pid int) string {
	out, err := exec.Command("ps", "-p", strconv.Itoa(pid), "-o", "args=").Output()
	if err != nil {
		return "(unknown)"
	}
	cmdline := strings.TrimSpace(string(out))
	if cmdline == "" {
		return "(unknown)"
	}
	return cmdline
}
