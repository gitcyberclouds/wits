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

// Guards the regression where the tool's own layout newlines (e.g. the blank
// line before "Source") were passed as Printer args and escaped to a literal
// "\n". The fixture has no newline in any value, so any literal "\n" in the
// output is a layout newline that leaked.
func TestRenderStandardDoesNotEscapeLayoutNewlines(t *testing.T) {
	for _, color := range []bool{false, true} {
		var buf bytes.Buffer
		RenderStandard(&buf, fixedFixture(), color, false)
		if strings.Contains(buf.String(), `\n`) {
			t.Errorf("color=%v: layout newline escaped to a literal \\n:\n%s", color, buf.String())
		}
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

// TestRenderStandardRestarts verifies the Restarts row renders only when the
// managing system reports at least one restart, matching the documented
// example output.
func TestRenderStandardRestarts(t *testing.T) {
	t.Parallel()

	// Zero restarts: the row must be omitted (no noise for the common case).
	var zero bytes.Buffer
	RenderStandard(&zero, fixedFixture(), false, false)
	if strings.Contains(zero.String(), "Restarts") {
		t.Errorf("Restarts row should be omitted when RestartCount is 0; output:\n%s", zero.String())
	}

	// Non-zero restarts: the row appears with the count.
	res := fixedFixture()
	res.RestartCount = 3
	var got bytes.Buffer
	RenderStandard(&got, res, false, false)
	if !strings.Contains(got.String(), "Restarts    : 3") {
		t.Errorf("expected \"Restarts    : 3\" in output; got:\n%s", got.String())
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
