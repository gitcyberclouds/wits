//go:build linux

package proc

import (
	"fmt"
	"os"
	"strings"
)

// GetCmdline returns the command line for a given PID
func GetCmdline(pid int) string {
	cmdlineBytes, err := os.ReadFile(fmt.Sprintf("/proc/%d/cmdline", pid))
	if err != nil {
		return "(unknown)"
	}
	cmd := strings.ReplaceAll(string(cmdlineBytes), "\x00", " ")
	cmdline := strings.TrimSpace(cmd)
	if cmdline == "" {
		return "(unknown)"
	}
	return cmdline
}
