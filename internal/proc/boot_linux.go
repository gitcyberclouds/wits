//go:build linux

package proc

import (
	"bufio"
	"os"
	"strconv"
	"strings"
	"time"
)

func bootTime() time.Time {
	f, err := os.Open("/proc/stat")
	if err != nil {
		return time.Now()
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "btime") {
			parts := strings.Fields(line)
			if len(parts) < 2 {
				continue
			}
			sec, _ := strconv.ParseInt(parts[1], 10, 64)
			return time.Unix(sec, 0)
		}
	}
	return time.Now()
}

func ticksPerSecond() int {
	return 100 // Linux default; portable enough for now
}

// startTimeFromTicks converts a process start time expressed in clock ticks
// since boot (/proc/[pid]/stat field 22) into an absolute time. It divides
// before scaling to nanoseconds: the naive `ticks * time.Second / hz` overflows
// int64 once ticks·1e9 exceeds ~9.2e18, i.e. for any process on a host whose
// uptime exceeds ~2.9 years, which would yield a garbage (often negative) time.
func startTimeFromTicks(boot time.Time, startTicks int64, hz int) time.Time {
	if hz <= 0 {
		hz = 100
	}
	h := int64(hz)
	secs := startTicks / h
	nsec := (startTicks % h) * int64(time.Second) / h
	return boot.Add(time.Duration(secs)*time.Second + time.Duration(nsec))
}
