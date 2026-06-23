package output

import (
	"bytes"
	"fmt"
	"strings"
	"testing"

	"github.com/pranshuparmar/witr/pkg/model"
)

func TestPrintTreeColored(t *testing.T) {
	chain := []model.Process{{PID: 1, Command: "systemd"}, {PID: 1234, Command: "nginx"}}
	kids := []model.Process{{PID: 2000, Command: "worker"}}

	var buf bytes.Buffer
	PrintTree(&buf, chain, kids, true)
	out := buf.String()
	for _, want := range []string{"systemd", "nginx", "worker", "pid 1234"} {
		if !strings.Contains(out, want) {
			t.Errorf("colored tree missing %q:\n%s", want, out)
		}
	}

	// More than ten children collapses into a "... and N more" line.
	many := make([]model.Process, 15)
	for i := range many {
		many[i] = model.Process{PID: i + 10, Command: fmt.Sprintf("c%d", i)}
	}
	var m bytes.Buffer
	PrintTree(&m, chain, many, true)
	if !strings.Contains(m.String(), "and 5 more") {
		t.Errorf("expected colored tree truncation; got:\n%s", m.String())
	}
}
