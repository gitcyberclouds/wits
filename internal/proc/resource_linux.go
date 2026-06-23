//go:build linux

package proc

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"github.com/godbus/dbus/v5"
	"github.com/pranshuparmar/witr/pkg/model"
)

// GetResourceContext returns resource usage context for a process
func GetResourceContext(pid int) *model.ResourceContext {
	ctx := &model.ResourceContext{}
	ctx.PreventsSleep = checkPreventsSleep(pid)
	ctx.ThermalState = getThermalState()
	ctx.AppNapped = getAppNapped(pid)

	// Compute CPU% once and derive both the usage figure and the energy-impact
	// label from it, instead of recomputing CPU via a second (slower) `top`.
	if cpu, err := GetCPUPercent(pid, true); err == nil {
		ctx.CPUUsage = cpu
		ctx.EnergyImpact = energyImpactLabel(cpu)
	}
	return ctx
}

// thermal zone info from /sys/class/thermal
func getThermalState() string {

	path := "/sys/class/thermal/thermal_zone0/temp"
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return ""
	}
	readText, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	tempstr := strings.TrimSpace(string(readText))
	temp, err := strconv.Atoi(tempstr)
	if err != nil {
		return ""
	}
	tempC := temp / 1000
	switch {
	case tempC > 90:
		return fmt.Sprintf("Critical thermal pressure %d", tempC)
	case tempC > 70:
		return fmt.Sprintf("High thermal pressure %d", tempC)
	case tempC > 60:
		return fmt.Sprintf("Warm thermal state %d", tempC)
	default:
		return fmt.Sprintf("Normal thermal state %d", tempC)
	}
}

// checkPreventsSleep checks if a process has sleep prevention assertions
func checkPreventsSleep(pid int) bool {
	conn, err := dbus.SystemBus()
	if err != nil {
		return false
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	// org.freedesktop.login1.Manager.ListInhibitors returns a(ssssuu):
	// (what, who, why, mode, uid, pid). Reading it over D-Bus avoids forking
	// `systemd-inhibit --list` on every resource lookup. SystemBus() returns a
	// shared connection, so we don't close it here.
	var inhibitors []struct {
		What, Who, Why, Mode string
		UID, PID             uint32
	}
	obj := conn.Object("org.freedesktop.login1", dbus.ObjectPath("/org/freedesktop/login1"))
	if err := obj.CallWithContext(ctx, "org.freedesktop.login1.Manager.ListInhibitors", 0).Store(&inhibitors); err != nil {
		return false
	}
	for _, in := range inhibitors {
		if int(in.PID) != pid {
			continue
		}
		what := strings.ToLower(in.What)
		if strings.Contains(what, "sleep") || strings.Contains(what, "idle") || strings.Contains(what, "shutdown") {
			return true
		}
	}
	return false
}

// detect if process is in a stopped/suspended state
func getAppNapped(pid int) bool {
	statFile := fmt.Sprintf("/proc/%d/stat", pid)
	data, err := os.ReadFile(statFile)
	if err != nil {
		return false
	}

	dataStr := string(data)
	lastParenIndex := strings.LastIndex(dataStr, ")")
	if lastParenIndex == -1 || lastParenIndex+2 >= len(dataStr) {
		return false
	}

	rest := dataStr[lastParenIndex+2:]
	fields := strings.Fields(rest)
	if len(fields) < 1 {
		return false
	}

	state := fields[0]
	return state == "T" || state == "t"
}

// energyImpactLabel maps a CPU-usage percentage to a coarse energy-impact band.
func energyImpactLabel(cpu float64) string {
	switch {
	case cpu > 50:
		return "Very High"
	case cpu > 25:
		return "High"
	case cpu > 10:
		return "Medium"
	case cpu > 2:
		return "Low"
	case cpu > 0:
		return "Very Low"
	default:
		return ""
	}
}

func GetCPUPercent(pid int, usePs ...bool) (float64, error) {
	var cpu float64

	shouldUsePs := len(usePs) > 0 && usePs[0]

	if shouldUsePs {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()

		out, err := exec.CommandContext(ctx, "ps", "-p", strconv.Itoa(pid), "-o", "pcpu=").Output()
		if err != nil {
			return 0, err
		}

		cpuStr := strings.TrimSpace(string(out))
		if cpuStr == "" {
			return 0, fmt.Errorf("empty ps output")
		}

		cpu, err = strconv.ParseFloat(cpuStr, 64)
		if err != nil {
			return 0, err
		}
	} else {
		// Use top (default)
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()

		out, err := exec.CommandContext(ctx, "top", "-b", "-n", "1", "-p", strconv.Itoa(pid)).Output()
		if err != nil {
			return 0, err
		}

		lines := strings.Split(string(out), "\n")
		found := false

		for _, line := range lines {
			fields := strings.Fields(line)
			if len(fields) >= 9 && fields[0] == strconv.Itoa(pid) {
				cpuStr := strings.TrimSuffix(fields[8], "%")
				cpu, err = strconv.ParseFloat(cpuStr, 64)
				if err == nil {
					found = true
					break
				}
			}
		}

		if !found {
			return 0, fmt.Errorf("process not found in top output")
		}
	}

	return cpu, nil
}
