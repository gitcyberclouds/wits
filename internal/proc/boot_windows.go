//go:build windows

package proc

import (
	"strings"
	"time"
)

func bootTime() time.Time {
	out, err := runPowerShell("Get-CimInstance -ClassName Win32_OperatingSystem | Select-Object -ExpandProperty LastBootUpTime | Get-Date -Format 'yyyyMMddHHmmss'")
	if err != nil {
		return time.Now()
	}
	// Output format:
	// 20231025123456
	val := strings.TrimSpace(string(out))
	if len(val) < 14 {
		return time.Now()
	}
	// Parse 20231025123456
	t, err := time.ParseInLocation("20060102150405", val[:14], time.Local)
	if err != nil {
		return time.Now()
	}
	return t
}
