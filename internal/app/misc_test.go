package app

import (
	"bytes"
	"strings"
	"testing"

	"github.com/pranshuparmar/witr/internal/output"
	"github.com/pranshuparmar/witr/pkg/model"
)

// TestUseColor pins the color-gating contract: color is enabled only for an
// interactive terminal and is suppressed when the user opts out. A bytes.Buffer
// is never a terminal, so it must always come back false — this is what keeps
// escape codes out of piped/redirected output.
func TestUseColor(t *testing.T) {
	t.Parallel()

	if useColor(appFlags{}, &bytes.Buffer{}) {
		t.Error("a bytes.Buffer is not a terminal; color should be disabled")
	}
	if useColor(appFlags{noColor: true}, &bytes.Buffer{}) {
		t.Error("--no-color must disable color")
	}
}

// TestPrintDivider checks the multi-target divider carries the human-readable
// target label.
func TestPrintDivider(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	p := output.NewPrinter(&buf)
	printDivider(p, model.Target{Type: model.TargetPort, Value: "8080"}, false, false)

	if got := buf.String(); !strings.Contains(got, "port: 8080") {
		t.Errorf("divider missing target label; got %q", got)
	}
}
