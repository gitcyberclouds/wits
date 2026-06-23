//go:build linux

package source

import (
	"context"
	"fmt"
	"math"
	"os"
	"strconv"
	"strings"
	"time"

	sd "github.com/coreos/go-systemd/v22/dbus"
	"github.com/pranshuparmar/witr/pkg/model"
)

// dbusTimeout bounds each systemd D-Bus interaction so a hung bus can't stall
// witr. It is generous relative to a healthy bus (single-digit milliseconds).
const dbusTimeout = 2 * time.Second

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

	// The unit name comes for free from the process cgroup; description, unit
	// file, restart count and timer schedule are best-effort enrichment over
	// systemd's D-Bus API.
	unitName := getUnitNameFromCgroup(ancestry[len(ancestry)-1].PID)

	src := &model.Source{
		Type:    model.SourceSystemd,
		Name:    unitName,
		Details: map[string]string{},
	}
	enrichFromSystemd(src, unitName)
	return src
}

// enrichFromSystemd fills Description, UnitFile, NRestarts and (for timer-
// triggered services) the schedule via systemd's D-Bus API. Every step is
// best-effort: a missing bus, a permission error, or an unloaded unit just
// leaves the corresponding field empty rather than failing detection. This
// replaces forking `systemctl show` (2-3 processes per report) with a single
// short-lived D-Bus connection.
func enrichFromSystemd(src *model.Source, unitName string) {
	if unitName == "" {
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), dbusTimeout)
	defer cancel()

	conn, err := sd.NewSystemConnectionContext(ctx)
	if err != nil {
		return // no usable bus — keep the cgroup-derived unit name only
	}
	defer conn.Close()

	if unit, err := conn.GetUnitPropertiesContext(ctx, unitName); err == nil {
		src.Description = stringProp(unit, "Description")
		if fp := stringProp(unit, "FragmentPath"); fp != "" {
			src.UnitFile = fp
		} else if sp := stringProp(unit, "SourcePath"); sp != "" {
			src.UnitFile = sp
		}
	}

	if strings.HasSuffix(unitName, ".service") {
		if svc, err := conn.GetUnitTypePropertiesContext(ctx, unitName, "Service"); err == nil {
			src.Details["NRestarts"] = strconv.FormatUint(uint64(uint32Prop(svc, "NRestarts")), 10)
		}
		timerUnit := strings.TrimSuffix(unitName, ".service") + ".timer"
		if sched := timerSchedule(ctx, conn, timerUnit); sched != "" {
			src.Details["schedule"] = sched
		}
	}
}

// timerSchedule renders a "<spec>, last: …, next: …" line for a .timer unit,
// or "" when the timer isn't loaded.
func timerSchedule(ctx context.Context, conn *sd.Conn, timerUnit string) string {
	tp, err := conn.GetUnitTypePropertiesContext(ctx, timerUnit, "Timer")
	if err != nil {
		return ""
	}

	spec := calendarSpec(tp["TimersCalendar"])
	if spec == "" {
		spec = monotonicSpec(tp["TimersMonotonic"])
	}
	if spec == "" {
		return ""
	}

	parts := []string{spec}
	if last := usecToTime(uint64Prop(tp, "LastTriggerUSec")); !last.IsZero() {
		parts = append(parts, "last: "+formatRelativeTime(last))
	}
	if next := usecToTime(uint64Prop(tp, "NextElapseUSecRealtime")); !next.IsZero() {
		parts = append(parts, "next: "+formatRelativeTime(next))
	}
	return strings.Join(parts, ", ")
}

// calendarSpec extracts the calendar expression from a TimersCalendar value,
// which D-Bus delivers as an array of (base, spec, next): e.g. "*-*-* 06,18:00:00".
func calendarSpec(v interface{}) string {
	for _, e := range timerEntries(v) {
		if len(e) >= 2 {
			if spec, ok := e[1].(string); ok && spec != "" {
				return spec
			}
		}
	}
	return ""
}

// monotonicSpec renders a TimersMonotonic value (base, usec, next) as a human
// phrase like "every 1d" or "every boot + 15min".
func monotonicSpec(v interface{}) string {
	for _, e := range timerEntries(v) {
		if len(e) < 2 {
			continue
		}
		base, _ := e[0].(string)
		usec, _ := e[1].(uint64)
		if usec == 0 {
			continue
		}
		human := humanDuration(time.Duration(usec) * time.Microsecond)
		switch {
		case strings.HasPrefix(base, "OnBoot"):
			return "every boot + " + human
		case strings.HasPrefix(base, "OnUnitInactive"):
			return "every " + human + " after idle"
		default: // OnUnitActive, OnActive, OnStartup
			return "every " + human
		}
	}
	return ""
}

// timerEntries normalizes a TimersCalendar/TimersMonotonic D-Bus value into a
// slice of struct fields, tolerating any decoding shape it can't read.
func timerEntries(v interface{}) [][]interface{} {
	entries, _ := v.([][]interface{})
	return entries
}

func humanDuration(d time.Duration) string {
	switch {
	case d >= 24*time.Hour:
		days := int(d / (24 * time.Hour))
		if hrs := int(d % (24 * time.Hour) / time.Hour); hrs > 0 {
			return fmt.Sprintf("%dd %dh", days, hrs)
		}
		return fmt.Sprintf("%dd", days)
	case d >= time.Hour:
		hrs := int(d / time.Hour)
		if mins := int(d % time.Hour / time.Minute); mins > 0 {
			return fmt.Sprintf("%dh %dmin", hrs, mins)
		}
		return fmt.Sprintf("%dh", hrs)
	case d >= time.Minute:
		mins := int(d / time.Minute)
		if secs := int(d % time.Minute / time.Second); secs > 0 {
			return fmt.Sprintf("%dmin %ds", mins, secs)
		}
		return fmt.Sprintf("%dmin", mins)
	default:
		return fmt.Sprintf("%ds", int(d.Seconds()))
	}
}

// usecToTime converts a systemd microseconds-since-epoch value to a time,
// treating 0 and the uint64 "infinity" sentinel as "no value".
func usecToTime(usec uint64) time.Time {
	if usec == 0 || usec == math.MaxUint64 {
		return time.Time{}
	}
	return time.UnixMicro(int64(usec))
}

func stringProp(m map[string]interface{}, key string) string {
	s, _ := m[key].(string)
	return s
}

func uint32Prop(m map[string]interface{}, key string) uint32 {
	n, _ := m[key].(uint32)
	return n
}

func uint64Prop(m map[string]interface{}, key string) uint64 {
	n, _ := m[key].(uint64)
	return n
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
