package tui

import (
	"sort"
	"testing"
	"time"

	"github.com/pranshuparmar/witr/pkg/model"
)

func TestFormatBytes(t *testing.T) {
	tests := []struct {
		in   uint64
		want string
	}{
		{0, "0 B"},
		{512, "512 B"},
		{1024, "1.0 KB"},
		{1536, "1.5 KB"},
		{1048576, "1.0 MB"},
		{1073741824, "1.0 GB"},
	}
	for _, tt := range tests {
		if got := formatBytes(tt.in); got != tt.want {
			t.Errorf("formatBytes(%d) = %q, want %q", tt.in, got, tt.want)
		}
	}
}

func TestTruncate(t *testing.T) {
	tests := []struct {
		s    string
		n    int
		want string
	}{
		{"hello", 10, "hello"},
		{"hello", 5, "hello"},
		{"hello", 3, "he…"},
		{"hello", 1, "…"},
		{"hello", 0, "…"},
	}
	for _, tt := range tests {
		if got := truncate(tt.s, tt.n); got != tt.want {
			t.Errorf("truncate(%q, %d) = %q, want %q", tt.s, tt.n, got, tt.want)
		}
	}
}

func TestTruncateMiddle(t *testing.T) {
	tests := []struct {
		s    string
		n    int
		want string
	}{
		{"short", 10, "short"},
		{"/usr/local/bin/witr", 11, "/usr/…/witr"},
		{"abcdefgh", 5, "abcd…"}, // n < 8 falls back to head truncation
		{"abc", 1, "…"},
	}
	for _, tt := range tests {
		if got := truncateMiddle(tt.s, tt.n); got != tt.want {
			t.Errorf("truncateMiddle(%q, %d) = %q, want %q", tt.s, tt.n, got, tt.want)
		}
	}
}

func TestCenterHeader(t *testing.T) {
	tests := []struct {
		title string
		width int
		want  string
	}{
		{"PID", 9, "   PID"}, // 6 pad, 3 left
		{"PID", 3, "PID"},    // width == title width: no pad
		{"toolong", 3, "toolong"},
		{"X", 2, "X"}, // 1 pad, 0 left
	}
	for _, tt := range tests {
		if got := centerHeader(tt.title, tt.width); got != tt.want {
			t.Errorf("centerHeader(%q, %d) = %q, want %q", tt.title, tt.width, got, tt.want)
		}
	}
}

func TestMergeLocksAndOpenFiles(t *testing.T) {
	locks := []*model.LockedFile{{PID: 1, Path: "/a"}, {PID: 2, Path: "/b"}}
	opens := []*model.LockedFile{{PID: 1, Path: "/a"}, {PID: 3, Path: "/c"}}

	merged := mergeLocksAndOpenFiles(locks, opens)
	// {1,/a} is in both; it must appear once. {3,/c} is open-only and added.
	if len(merged) != 3 {
		t.Fatalf("merged length = %d, want 3 (got %v)", len(merged), merged)
	}
	// Locks come first and verbatim; the duplicate open is dropped, /c kept.
	want := []struct {
		pid  int
		path string
	}{{1, "/a"}, {2, "/b"}, {3, "/c"}}
	for i, w := range want {
		if merged[i].PID != w.pid || merged[i].Path != w.path {
			t.Errorf("merged[%d] = {%d,%q}, want {%d,%q}", i, merged[i].PID, merged[i].Path, w.pid, w.path)
		}
	}
}

func TestProcessMatches(t *testing.T) {
	p := model.Process{Command: "nginx", PID: 1234, User: "root", Cmdline: "nginx -g daemon off"}
	tests := []struct {
		filter string // already lowercased, as the caller passes it
		want   bool
	}{
		{"nginx", true},  // command
		{"1234", true},   // pid
		{"root", true},   // user
		{"daemon", true}, // cmdline
		{"redis", false}, // no match
		{"NGINX", false}, // caller lowercases; an uppercase filter never matches
	}
	for _, tt := range tests {
		if got := processMatches(p, tt.filter); got != tt.want {
			t.Errorf("processMatches(p, %q) = %v, want %v", tt.filter, got, tt.want)
		}
	}
}

func TestLessPorts(t *testing.T) {
	a := model.OpenPort{Port: 80, Protocol: "tcp", Address: "0.0.0.0", State: "LISTEN"}
	b := model.OpenPort{Port: 443, Protocol: "udp", Address: "127.0.0.1", State: "ESTABLISHED"}
	tests := []struct {
		col  string
		want bool // lessPorts(a, b, col)
	}{
		{"port", true},    // 80 < 443
		{"proto", true},   // "tcp" < "udp"
		{"addr", true},    // "0.0.0.0" < "127.0.0.1"
		{"state", false},  // "listen" < "established" is false
		{"unknown", true}, // falls back to port: 80 < 443
	}
	for _, tt := range tests {
		if got := lessPorts(a, b, tt.col); got != tt.want {
			t.Errorf("lessPorts(a, b, %q) = %v, want %v", tt.col, got, tt.want)
		}
	}
}

func TestProcessSorter(t *testing.T) {
	procs := []model.Process{
		{PID: 30, CPUPercent: 5.0, MemoryRSS: 100},
		{PID: 10, CPUPercent: 9.0, MemoryRSS: 300},
		{PID: 20, CPUPercent: 5.0, MemoryRSS: 200},
	}

	pidsAfter := func(col string, desc bool) []int {
		cp := make([]model.Process, len(procs))
		copy(cp, procs)
		sort.Stable(processSorter{procs: cp, col: col, desc: desc})
		out := make([]int, len(cp))
		for i, p := range cp {
			out[i] = p.PID
		}
		return out
	}

	if got := pidsAfter("pid", false); !equalInts(got, []int{10, 20, 30}) {
		t.Errorf("pid asc = %v, want [10 20 30]", got)
	}
	if got := pidsAfter("cpu", true); !equalInts(got, []int{10, 30, 20}) {
		// CPU desc: 9.0 first; the two 5.0 tie, and desc inverts the PID
		// tiebreak too, so 30 precedes 20.
		t.Errorf("cpu desc = %v, want [10 30 20]", got)
	}
	if got := pidsAfter("mem", false); !equalInts(got, []int{30, 20, 10}) {
		t.Errorf("mem asc = %v, want [30 20 10]", got)
	}
}

func TestAdjustRefreshInterval(t *testing.T) {
	s := time.Second
	type step struct {
		took     time.Duration
		interval time.Duration
		slow     int
		fast     int
	}
	// Start at the base 3s; walk a slow spell up, then a fast spell back down.
	steps := []step{
		{2 * s, 3 * s, 1, 0}, // 67% of 3s -> slow, streak 1 (no change yet)
		{2 * s, 6 * s, 0, 0}, // slow again -> grow to 6s, reset
		{2 * s, 6 * s, 0, 0}, // 33% of 6s -> stable band, reset
		{1 * s, 6 * s, 0, 1}, // 17% of 6s -> fast, streak 1
		{1 * s, 3 * s, 0, 0}, // fast again -> shrink to 3s (floor), reset
		{1 * s, 3 * s, 0, 0}, // 33% of 3s -> stable, already at floor
	}
	interval, slow, fast := 3*s, 0, 0
	for i, st := range steps {
		interval, slow, fast = adjustRefreshInterval(interval, st.took, slow, fast)
		if interval != st.interval || slow != st.slow || fast != st.fast {
			t.Errorf("step %d (took %v): got (%v, %d, %d), want (%v, %d, %d)",
				i, st.took, interval, slow, fast, st.interval, st.slow, st.fast)
		}
	}

	// Ceiling: a second slow sample at the cap stays at the cap.
	if got, _, _ := adjustRefreshInterval(maxRefreshInterval, 25*s, 1, 0); got != maxRefreshInterval {
		t.Errorf("ceiling: got %v, want %v", got, maxRefreshInterval)
	}
}

func equalInts(a, b []int) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
