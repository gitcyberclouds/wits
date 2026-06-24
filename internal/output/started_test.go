package output

import (
	"testing"
	"time"
)

func TestFormatStartedAt(t *testing.T) {
	if rel, abs := FormatStartedAt(time.Time{}); rel != "unknown" || abs != "" {
		t.Errorf("zero time = (%q, %q), want (\"unknown\", \"\")", rel, abs)
	}

	now := time.Now()
	cases := []struct {
		ago  time.Duration
		want string
	}{
		{30 * time.Second, "just now"},
		{5 * time.Minute, "5 min ago"},
		{90 * time.Minute, "1 hour ago"},
		{5 * time.Hour, "5 hours ago"},
		{30 * time.Hour, "1 day ago"},
		{72 * time.Hour, "3 days ago"},
	}
	for _, tc := range cases {
		rel, abs := FormatStartedAt(now.Add(-tc.ago))
		if rel != tc.want {
			t.Errorf("FormatStartedAt(-%v) rel = %q, want %q", tc.ago, rel, tc.want)
		}
		if abs == "" {
			t.Errorf("FormatStartedAt(-%v) absolute timestamp should not be empty", tc.ago)
		}
	}
}
