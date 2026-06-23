//go:build linux

package source

import (
	"os"
	"testing"
	"time"
)

func TestExtractTimerSpec(t *testing.T) {
	tests := []struct {
		raw  string
		want string
	}{
		{"", ""},
		{"{ OnCalendar=*-*-* 06,18:00:00 ; next_elapse=Mon }", "*-*-* 06,18:00:00"},
		{"{ OnUnitActiveUSec=1d ; next_elapse=x }", "every 1d"},
		{"{ OnBootUSec=15min ; next_elapse=x }", "every boot + 15min"},
		{"{ OnUnitInactiveUSec=5min ; x }", "every 5min after idle"},
		{"{ Unknown=foo }", ""},
	}
	for _, tt := range tests {
		if got := extractTimerSpec(tt.raw); got != tt.want {
			t.Errorf("extractTimerSpec(%q) = %q, want %q", tt.raw, got, tt.want)
		}
	}
}

func TestFormatRelativeTime(t *testing.T) {
	now := time.Now()
	// Durations are buffered off the truncation boundaries so sub-second drift
	// between now and the time.Since() call inside the function can't flip an
	// "N min" bucket to "N-1".
	cases := []struct {
		d    time.Duration // offset from now; negative = past, positive = future
		want string
	}{
		{-30 * time.Second, "<1 min ago"},
		{-(5*time.Minute + 30*time.Second), "5 min ago"},
		{-(3*time.Hour + 30*time.Minute), "3h ago"},
		{-50 * time.Hour, "2d ago"},
		{30 * time.Second, "in <1 min"},
		{5*time.Minute + 30*time.Second, "in 5 min"},
		{3*time.Hour + 30*time.Minute, "in 3h"},
		{50 * time.Hour, "in 2d"},
	}
	for _, tc := range cases {
		if got := formatRelativeTime(now.Add(tc.d)); got != tc.want {
			t.Errorf("formatRelativeTime(now%+v) = %q, want %q", tc.d, got, tc.want)
		}
	}
}

func TestGetUnitNameFromCgroupSelf(t *testing.T) {
	// The result depends on the host's cgroup layout (a .scope, a .service, or
	// nothing), so we only exercise the read+parse without asserting a value.
	_ = getUnitNameFromCgroup(os.Getpid())

	// PID 0 has no cgroup file, so the read fails and we get "".
	if got := getUnitNameFromCgroup(0); got != "" {
		t.Errorf("getUnitNameFromCgroup(0) = %q, want empty", got)
	}
}
