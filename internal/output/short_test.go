package output

import (
	"bytes"
	"strings"
	"testing"

	"github.com/pranshuparmar/witr/pkg/model"
)

// shortFixture exercises every branch of RenderShort: a first node (no
// separator), middle nodes (with arrow separator), and a final node (last
// element of the chain).
func shortFixture() model.Result {
	return model.Result{
		Ancestry: []model.Process{
			{PID: 1, Command: "systemd"},
			{PID: 1234, Command: "bash"},
			{PID: 5678, Command: "nginx"},
		},
	}
}

func TestRenderShort(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	RenderShort(&buf, shortFixture(), false)
	got := strings.TrimSpace(buf.String())

	want := "systemd (pid 1) → bash (pid 1234) → nginx (pid 5678)"
	if got != want {
		t.Errorf("RenderShort = %q, want %q", got, want)
	}
}

func TestRenderShortSingleProcess(t *testing.T) {
	t.Parallel()

	r := model.Result{Ancestry: []model.Process{{PID: 1, Command: "init"}}}

	var buf bytes.Buffer
	RenderShort(&buf, r, false)
	got := strings.TrimSpace(buf.String())

	want := "init (pid 1)"
	if got != want {
		t.Errorf("RenderShort single process = %q, want %q", got, want)
	}
}

func TestRenderShortEmptyAncestry(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	RenderShort(&buf, model.Result{}, false)

	// Empty ancestry should produce just a newline, not panic or omit anything.
	if got := buf.String(); got != "\n" {
		t.Errorf("RenderShort with empty ancestry = %q, want a single newline", got)
	}
}
