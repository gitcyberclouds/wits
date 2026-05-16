package output

import (
	"bytes"
	"strings"
	"testing"
	"time"

	"github.com/pranshuparmar/witr/pkg/model"
)

// fixedFixture returns a deterministic Result so RenderStandard's output can
// be asserted line-by-line. Timestamps are far enough in the past for the
// "Started" relative phrasing to be stable ("X days ago" branch).
func fixedFixture() model.Result {
	startedAt := time.Date(2024, 1, 1, 10, 0, 0, 0, time.UTC)
	target := model.Process{
		PID:       1234,
		PPID:      1,
		Command:   "nginx",
		Cmdline:   "/usr/sbin/nginx -g daemon off;",
		User:      "nginx",
		StartedAt: startedAt,
		Service:   "nginx.service",
		Sockets: []model.Socket{
			{Address: "192.168.1.5", Port: 80, Protocol: "TCP", State: "ESTABLISHED"},
			{Address: "0.0.0.0", Port: 443, Protocol: "TCP", State: "LISTEN"},
			{Address: "0.0.0.0", Port: 443, Protocol: "TCP", State: "ESTABLISHED"},
			{Address: "0.0.0.0", Port: 80, Protocol: "TCP", State: "LISTEN"},
		},
		Health: "healthy",
		Forked: "not-forked",
	}
	return model.Result{
		Target:   model.Target{Type: model.TargetName, Value: "nginx"},
		Process:  target,
		Ancestry: []model.Process{{PID: 1, Command: "systemd"}, target},
		Source:   model.Source{Type: model.SourceSystemd, Name: "nginx.service"},
	}
}

// TestRenderStandardContract pins down the structural contract of the
// standard output: which sections appear, their order, and the format of
// rows users (and scripts piping our output) depend on. We assert on
// substrings rather than exact byte-for-byte equality so the test doesn't
// break on cosmetic tweaks like extra blank lines or label padding.
func TestRenderStandardContract(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	RenderStandard(&buf, fixedFixture(), false, false)
	out := buf.String()

	mustContain := []string{
		"Target      : nginx",
		"Process     : nginx (pid 1234)",
		"User        : nginx",
		"Service     : nginx.service",
		"Command     : /usr/sbin/nginx -g daemon off;",
		"Why It Exists :",
		"systemd",
		"nginx",
		"Source      : nginx.service (systemd)",
		"Sockets     :",
		"0.0.0.0:80 (TCP | LISTENING)",
		"0.0.0.0:443 (TCP | LISTENING)",
		"0.0.0.0:443 (TCP | ESTABLISHED)",
		"192.168.1.5:80 (TCP | ESTABLISHED)",
	}
	for _, s := range mustContain {
		if !strings.Contains(out, s) {
			t.Errorf("RenderStandard output missing %q\n---\n%s\n---", s, out)
		}
	}
}

// TestRenderStandardSocketsOrder verifies the Sockets section actually emits
// rows in sortSockets order: addresses grouped, LISTEN above ESTABLISHED on
// shared address:port, ports ascending. This is the renderer-level mirror of
// TestSortSockets — it catches regressions where the comparator is correct
// but the renderer accidentally re-orders the slice afterwards.
func TestRenderStandardSocketsOrder(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	RenderStandard(&buf, fixedFixture(), false, false)
	out := buf.String()

	rows := []string{
		"0.0.0.0:80 (TCP | LISTENING)",
		"0.0.0.0:443 (TCP | LISTENING)",
		"0.0.0.0:443 (TCP | ESTABLISHED)",
		"192.168.1.5:80 (TCP | ESTABLISHED)",
	}
	last := -1
	for _, row := range rows {
		idx := strings.Index(out, row)
		if idx == -1 {
			t.Fatalf("missing socket row %q in output", row)
		}
		if idx < last {
			t.Errorf("socket row %q appeared before the previous row (idx %d < %d)", row, idx, last)
		}
		last = idx
	}
}

// TestRenderStandardOmitsEmptyOptionalSections protects against accidentally
// printing labels with no value (e.g. "Container :"). The fixture leaves
// Container, GitRepo, and warnings empty; none of those labels should appear.
func TestRenderStandardOmitsEmptyOptionalSections(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	RenderStandard(&buf, fixedFixture(), false, false)
	out := buf.String()

	mustNotContain := []string{
		"Container   :",
		"Git Repo    :",
		"Warnings    :",
	}
	for _, s := range mustNotContain {
		if strings.Contains(out, s) {
			t.Errorf("RenderStandard output should not contain %q for empty fixture; output:\n%s", s, out)
		}
	}
}
