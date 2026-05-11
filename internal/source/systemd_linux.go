//go:build linux

package source

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/pranshuparmar/witr/pkg/model"
)

// IsSystemdRunning checks whether systemd is actually the running init system.
// This is the canonical check used by sd_booted() in libsystemd.
func IsSystemdRunning() bool {
	_, err := os.Stat("/run/systemd/system")
	return err == nil
}

func detectSystemd(ancestry []model.Process) *model.Source {
	// Verify systemd is actually the init system, not just that PID 1
	// happens to be named "init" (which could be SysVinit, OpenRC, runit, etc.)
	if !IsSystemdRunning() {
		return nil
	}

	hasPID1 := false
	for _, p := range ancestry {
		if p.PID == 1 {
			hasPID1 = true
			break
		}
	}

	if !hasPID1 {
		return nil
	}

	targetProc := ancestry[len(ancestry)-1]
	props := resolveSystemdProperties(targetProc.PID)

	// Keep only supplemental details (top-level fields already hold name/desc/unitfile)
	details := map[string]string{}
	if v := props["NRestarts"]; v != "" {
		details["NRestarts"] = v
	}

	src := &model.Source{
		Type:        model.SourceSystemd,
		Name:        props["UnitName"],
		Description: props["Description"],
		UnitFile:    props["UnitFile"],
		Details:     details,
	}

	// Check if the service is triggered by a systemd timer
	if unitName := props["UnitName"]; strings.HasSuffix(unitName, ".service") {
		timerUnit := strings.TrimSuffix(unitName, ".service") + ".timer"
		if schedule := resolveTimerSchedule(timerUnit); schedule != "" {
			src.Details["schedule"] = schedule
		}
	}

	return src
}

// resolveSystemdProperties fetches Description, FragmentPath/SourcePath, and NRestarts
// in a single systemctl call to avoid spawning multiple processes.
func resolveSystemdProperties(pid int) map[string]string {
	result := map[string]string{}

	if _, err := exec.LookPath("systemctl"); err != nil {
		return result
	}

	unitName := getUnitNameFromCgroup(pid)
	if unitName != "" {
		result["UnitName"] = unitName
	}

	// Try cgroup-resolved unit name first, fall back to PID-based lookup
	targets := []string{}
	if unitName != "" {
		targets = append(targets, unitName)
	}
	targets = append(targets, fmt.Sprintf("%d", pid))

	props := []string{"Description", "FragmentPath", "SourcePath", "NRestarts"}

	for _, target := range targets {
		values := querySystemdProperties(props, target)

		if result["Description"] == "" && values["Description"] != "" {
			result["Description"] = values["Description"]
		}
		if result["UnitFile"] == "" {
			if values["FragmentPath"] != "" {
				result["UnitFile"] = values["FragmentPath"]
			} else if values["SourcePath"] != "" {
				result["UnitFile"] = values["SourcePath"]
			}
		}
		if result["NRestarts"] == "" && values["NRestarts"] != "" {
			result["NRestarts"] = values["NRestarts"]
		}

		// Stop once we have all the info we need
		if result["Description"] != "" && result["UnitFile"] != "" && result["NRestarts"] != "" {
			break
		}
	}

	return result
}

// querySystemdProperties fetches multiple properties in a single systemctl invocation.
func querySystemdProperties(props []string, target string) map[string]string {
	args := []string{"show"}
	for _, p := range props {
		args = append(args, "-p", p)
	}
	args = append(args, "--", target)

	cmd := exec.Command("systemctl", args...)
	out, err := cmd.Output()
	if err != nil {
		return nil
	}

	result := make(map[string]string, len(props))
	for _, line := range strings.Split(strings.TrimSpace(string(out)), "\n") {
		k, v, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		v = strings.TrimSpace(v)
		if v == "" || strings.Contains(v, "not set") {
			continue
		}
		result[k] = v
	}
	return result
}

// resolveTimerSchedule checks if a .timer unit exists and extracts schedule info.
func resolveTimerSchedule(timerUnit string) string {
	props := querySystemdProperties(
		[]string{"TimersCalendar", "TimersMonotonic", "LastTriggerUSec", "NextElapseUSecRealtime"},
		timerUnit,
	)
	if props == nil {
		return ""
	}

	// Extract schedule spec from calendar or monotonic timer
	schedule := extractTimerSpec(props["TimersCalendar"])
	if schedule == "" {
		schedule = extractTimerSpec(props["TimersMonotonic"])
	}
	if schedule == "" {
		return ""
	}

	var parts []string
	parts = append(parts, schedule)

	if last := props["LastTriggerUSec"]; last != "" && last != "n/a" {
		if t, err := time.Parse("Mon 2006-01-02 15:04:05 MST", last); err == nil {
			parts = append(parts, "last: "+formatRelativeTime(t))
		}
	}
	if next := props["NextElapseUSecRealtime"]; next != "" && next != "n/a" {
		if t, err := time.Parse("Mon 2006-01-02 15:04:05 MST", next); err == nil {
			parts = append(parts, "next: "+formatRelativeTime(t))
		}
	}

	return strings.Join(parts, ", ")
}

// extractTimerSpec parses the value from systemd's TimersCalendar or TimersMonotonic format.
// Format examples:
//
//	"{ OnCalendar=*-*-* 06,18:00:00 ; next_elapse=... }"
//	"{ OnUnitActiveUSec=1d ; next_elapse=... }"
//	"{ OnBootUSec=15min ; next_elapse=... }"
func extractTimerSpec(raw string) string {
	if raw == "" {
		return ""
	}

	// Look for the first key=value pair inside braces
	for _, prefix := range []string{"OnCalendar=", "OnUnitActiveUSec=", "OnBootUSec=", "OnUnitInactiveUSec="} {
		idx := strings.Index(raw, prefix)
		if idx == -1 {
			continue
		}
		after := raw[idx+len(prefix):]
		// Trim at the next semicolon or closing brace
		if semi := strings.IndexAny(after, ";}"); semi != -1 {
			after = after[:semi]
		}
		after = strings.TrimSpace(after)
		if after == "" {
			continue
		}
		// Prefix monotonic timers with the trigger type for clarity
		switch {
		case strings.HasPrefix(prefix, "OnBoot"):
			return "every boot + " + after
		case strings.HasPrefix(prefix, "OnUnitActive"):
			return "every " + after
		case strings.HasPrefix(prefix, "OnUnitInactive"):
			return "every " + after + " after idle"
		default:
			return after
		}
	}
	return ""
}

// formatRelativeTime returns a human-friendly relative time string.
func formatRelativeTime(t time.Time) string {
	d := time.Since(t)
	if d < 0 {
		d = -d
		switch {
		case d < time.Minute:
			return "in <1 min"
		case d < time.Hour:
			return fmt.Sprintf("in %d min", int(d.Minutes()))
		case d < 24*time.Hour:
			return fmt.Sprintf("in %dh", int(d.Hours()))
		default:
			return fmt.Sprintf("in %dd", int(d.Hours()/24))
		}
	}
	switch {
	case d < time.Minute:
		return "<1 min ago"
	case d < time.Hour:
		return fmt.Sprintf("%d min ago", int(d.Minutes()))
	case d < 24*time.Hour:
		return fmt.Sprintf("%dh ago", int(d.Hours()))
	default:
		return fmt.Sprintf("%dd ago", int(d.Hours()/24))
	}
}

func getUnitNameFromCgroup(pid int) string {
	data, err := os.ReadFile(fmt.Sprintf("/proc/%d/cgroup", pid))
	if err != nil {
		return ""
	}

	lines := strings.Split(string(data), "\n")
	for _, line := range lines {
		parts := strings.SplitN(line, ":", 3)
		if len(parts) < 3 {
			continue
		}
		controllers := parts[1]
		path := parts[2]

		if controllers == "" || strings.Contains(controllers, "systemd") {
			path = strings.TrimSpace(path)
			pathParts := strings.Split(path, "/")

			for i := len(pathParts) - 1; i >= 0; i-- {
				part := pathParts[i]
				if strings.HasSuffix(part, ".service") || strings.HasSuffix(part, ".scope") {
					return part
				}
			}
		}
	}
	return ""
}
