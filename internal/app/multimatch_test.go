package app

import (
	"bytes"
	"os"
	"strings"
	"testing"

	"github.com/pranshuparmar/witr/internal/output"
	"github.com/pranshuparmar/witr/pkg/model"
)

func TestPrintMultiMatch(t *testing.T) {
	for _, color := range []bool{false, true} {
		var buf bytes.Buffer
		outp := output.NewPrinter(&buf)
		// Anchor on our own PID so ReadProcess succeeds deterministically.
		printMultiMatch(outp, []int{os.Getpid()}, color, "witr --pid 1234")
		out := buf.String()
		if !strings.Contains(out, "Multiple matching processes") || !strings.Contains(out, "witr --pid 1234") {
			t.Errorf("color=%v: printMultiMatch output wrong:\n%s", color, out)
		}
		if strings.Contains(out, `\n`) {
			t.Errorf("color=%v: layout newline escaped to a literal \\n:\n%s", color, out)
		}
	}
}

func TestPrintContainerMultiMatch(t *testing.T) {
	matches := []*model.ContainerMatch{
		{Name: "web", Image: "nginx:latest", Runtime: "docker", Status: "Up 3 min", Ports: "0.0.0.0:80->80/tcp"},
	}
	for _, color := range []bool{false, true} {
		var buf bytes.Buffer
		outp := output.NewPrinter(&buf)
		printContainerMultiMatch(outp, matches, color)
		out := buf.String()
		if !strings.Contains(out, "Multiple matching containers") || !strings.Contains(out, "web") || !strings.Contains(out, "nginx:latest") {
			t.Errorf("color=%v: printContainerMultiMatch output wrong:\n%s", color, out)
		}
		if strings.Contains(out, `\n`) {
			t.Errorf("color=%v: layout newline escaped to a literal \\n:\n%s", color, out)
		}
	}
}
