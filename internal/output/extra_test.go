package output

import (
	"bytes"
	"strings"
	"testing"

	"github.com/pranshuparmar/witr/pkg/model"
)

func TestRenderWarnings(t *testing.T) {
	r := model.Result{
		Process:  model.Process{PID: 1234, Command: "nginx"},
		Ancestry: []model.Process{{PID: 1, Command: "systemd"}, {PID: 1234, Command: "nginx"}},
		Warnings: []string{"Process is running as root"},
	}

	for _, color := range []bool{false, true} {
		var buf bytes.Buffer
		RenderWarnings(&buf, r, color)
		out := buf.String()
		if !strings.Contains(out, "Warnings") || !strings.Contains(out, "running as root") {
			t.Errorf("color=%v: warnings-only render wrong:\n%s", color, out)
		}
	}

	t.Run("no warnings", func(t *testing.T) {
		r2 := r
		r2.Warnings = nil
		var buf bytes.Buffer
		RenderWarnings(&buf, r2, false)
		if !strings.Contains(buf.String(), "No warnings.") {
			t.Errorf("expected the no-warnings message; got:\n%s", buf.String())
		}
	})
}

func TestFormatContainerLine(t *testing.T) {
	match := &model.ContainerMatch{Runtime: "docker", ID: "abc123def456", Name: "web", Image: "nginx:latest"}
	line := FormatContainerLine(match)
	if line == "" || !strings.Contains(line, "web") {
		t.Errorf("FormatContainerLine = %q, want a non-empty line naming the container", line)
	}
}

func TestFormatDetailLabel(t *testing.T) {
	// A known key maps to its padded display label.
	if got := formatDetailLabel("type"); !strings.Contains(got, "Type") {
		t.Errorf("formatDetailLabel(\"type\") = %q, want it to contain \"Type\"", got)
	}
	// An unknown key is padded and passed through verbatim.
	if got := formatDetailLabel("custom-key"); !strings.Contains(got, "custom-key") {
		t.Errorf("formatDetailLabel(\"custom-key\") = %q, want it to contain the key", got)
	}
}
