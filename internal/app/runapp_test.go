package app

import (
	"bytes"
	"os"
	"strconv"
	"strings"
	"testing"
)

// TestRunAppRendersReportForSelf drives the full command in-process against our
// own PID. The existing exit-code tests run the built binary as a subprocess
// (so they don't show up in coverage) and only exercise error paths; this is
// the one in-process test of the *success* path: runApp -> processTarget ->
// renderResult producing a real report.
func TestRunAppRendersReportForSelf(t *testing.T) {
	pid := strconv.Itoa(os.Getpid())

	// runApp reads os.Args directly to preserve command-line target ordering,
	// so it must agree with the args we hand cobra.
	oldArgs := os.Args
	os.Args = []string{"witr", "--pid", pid}
	t.Cleanup(func() { os.Args = oldArgs })

	cmd := Root()
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs([]string{"--pid", pid})
	t.Cleanup(func() { cmd.SetArgs(nil) })

	// Execute may return a non-nil error when the process carries warnings
	// (exit 1) — that's fine, we're asserting the report rendered, not the code.
	_ = cmd.Execute()

	report := out.String()
	if !strings.Contains(report, "Process") || !strings.Contains(report, pid) {
		t.Errorf("runApp did not render a report for self (pid %s):\n%s", pid, report)
	}
}
