//go:build linux

package proc

import (
	"strings"
	"testing"

	"github.com/pranshuparmar/witr/pkg/model"
)

func TestMapTCPState(t *testing.T) {
	known := map[int]string{
		1: "ESTABLISHED", 2: "SYN_SENT", 6: "TIME_WAIT", 10: "LISTEN", 11: "CLOSING",
	}
	for in, want := range known {
		if got := mapTCPState(in); got != want {
			t.Errorf("mapTCPState(%d) = %q, want %q", in, got, want)
		}
	}
	if got := mapTCPState(99); !strings.HasPrefix(got, "UNKNOWN") {
		t.Errorf("mapTCPState(99) = %q, want an UNKNOWN fallback", got)
	}
}

func TestIsProblematicState(t *testing.T) {
	for _, s := range []string{"TIME_WAIT", "CLOSE_WAIT", "FIN_WAIT_1", "FIN_WAIT_2"} {
		if !isProblematicState(s) {
			t.Errorf("%q should be flagged problematic", s)
		}
	}
	for _, s := range []string{"LISTEN", "ESTABLISHED", "SYN_SENT"} {
		if isProblematicState(s) {
			t.Errorf("%q should not be flagged problematic", s)
		}
	}
}

func TestAddStateExplanation(t *testing.T) {
	// A state with a documented workaround gets both fields.
	tw := &model.SocketInfo{State: "TIME_WAIT"}
	addStateExplanation(tw)
	if tw.Explanation == "" || tw.Workaround == "" {
		t.Errorf("TIME_WAIT: explanation=%q workaround=%q, want both set", tw.Explanation, tw.Workaround)
	}
	// A benign state gets an explanation but no workaround.
	ln := &model.SocketInfo{State: "LISTEN"}
	addStateExplanation(ln)
	if ln.Explanation == "" {
		t.Error("LISTEN should have an explanation")
	}
	// An unknown state falls back to a generic explanation naming the state.
	unk := &model.SocketInfo{State: "WEIRD"}
	addStateExplanation(unk)
	if !strings.Contains(unk.Explanation, "WEIRD") {
		t.Errorf("unknown-state explanation = %q, want it to mention the state", unk.Explanation)
	}
}
