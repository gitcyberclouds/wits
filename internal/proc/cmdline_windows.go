//go:build windows

package proc

import (
	"fmt"
	"strings"
)

// GetCmdline returns the command line for a given PID. Bounded by a short
// PowerShell timeout so a stalled WMI provider can't hang the caller.
func GetCmdline(pid int) string {
	out, err := runPowerShell(fmt.Sprintf("Get-CimInstance -ClassName Win32_Process -Filter \"ProcessId=%d\" | Select-Object -ExpandProperty CommandLine", pid))
	if err != nil {
		return "(unknown)"
	}
	val := strings.TrimSpace(string(out))
	if val == "" {
		return "(unknown)"
	}
	return val
}
