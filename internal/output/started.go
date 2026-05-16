package output

import (
	"fmt"
	"time"
)

// FormatStartedAt returns the "Started" line components: a relative phrase
// like "2 days ago" and an absolute timestamp like "Mon 2026-05-15 09:42:11 +00:00".
// Empty time produces ("unknown", "").
func FormatStartedAt(t time.Time) (rel, absolute string) {
	if t.IsZero() {
		return "unknown", ""
	}
	dur := time.Since(t)
	switch {
	case dur.Hours() >= 48:
		rel = fmt.Sprintf("%d days ago", int(dur.Hours())/24)
	case dur.Hours() >= 24:
		rel = "1 day ago"
	case dur.Hours() >= 2:
		rel = fmt.Sprintf("%d hours ago", int(dur.Hours()))
	case dur.Minutes() >= 60:
		rel = "1 hour ago"
	default:
		mins := int(dur.Minutes())
		if mins > 0 {
			rel = fmt.Sprintf("%d min ago", mins)
		} else {
			rel = "just now"
		}
	}
	absolute = t.Format("Mon 2006-01-02 15:04:05 -07:00")
	return rel, absolute
}
