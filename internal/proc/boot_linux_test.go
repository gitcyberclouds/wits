//go:build linux

package proc

import (
	"testing"
	"time"
)

func TestStartTimeFromTicks(t *testing.T) {
	boot := time.Unix(1_600_000_000, 0)

	// Basic conversion: 250 ticks at 100 Hz is 2.5s after boot.
	if got := startTimeFromTicks(boot, 250, 100); !got.Equal(boot.Add(2500 * time.Millisecond)) {
		t.Errorf("startTimeFromTicks(boot, 250, 100) = %v, want boot+2.5s", got)
	}

	// hz <= 0 falls back to 100 rather than dividing by zero.
	if got := startTimeFromTicks(boot, 100, 0); !got.Equal(boot.Add(time.Second)) {
		t.Errorf("startTimeFromTicks(boot, 100, 0) = %v, want boot+1s", got)
	}

	// Multi-year uptime must not overflow int64. The naive ticks*time.Second/hz
	// overflows past ~2.9 years and lands before boot; the helper must not.
	bigTicks := int64(5 * 365 * 24 * 60 * 60 * 100) // ~5 years at 100 Hz
	got := startTimeFromTicks(boot, bigTicks, 100)
	if got.Before(boot) {
		t.Fatalf("startTimeFromTicks(big) = %v, before boot %v (overflow)", got, boot)
	}
	if d, want := got.Sub(boot), time.Duration(bigTicks/100)*time.Second; d != want {
		t.Errorf("startTimeFromTicks(big) = boot+%v, want boot+%v", d, want)
	}
}
