package output

import (
	"bytes"
	"strings"
	"testing"

	"github.com/pranshuparmar/witr/pkg/model"
)

func TestPrintTreeAncestryAndChildren(t *testing.T) {
	t.Parallel()

	chain := []model.Process{
		{PID: 1, Command: "systemd"},
		{PID: 100, Command: "nginx"},
	}
	children := []model.Process{
		{PID: 101, Command: "worker-1"},
		{PID: 102, Command: "worker-2"},
	}

	var buf bytes.Buffer
	PrintTree(&buf, chain, children, false)
	out := buf.String()

	mustContain := []string{
		"systemd (pid 1)",
		"└─ nginx (pid 100)",
		"├─ worker-1 (pid 101)",
		"└─ worker-2 (pid 102)",
	}
	for _, s := range mustContain {
		if !strings.Contains(out, s) {
			t.Errorf("PrintTree output missing %q\n---\n%s\n---", s, out)
		}
	}
}

func TestPrintTreeTruncatesChildrenOverLimit(t *testing.T) {
	t.Parallel()

	chain := []model.Process{{PID: 100, Command: "nginx"}}

	// 12 children — limit is 10, expect "... and 2 more"
	children := make([]model.Process, 12)
	for i := range children {
		children[i] = model.Process{PID: 200 + i, Command: "worker"}
	}

	var buf bytes.Buffer
	PrintTree(&buf, chain, children, false)
	out := buf.String()

	if !strings.Contains(out, "... and 2 more") {
		t.Errorf("PrintTree should report truncation; output:\n%s", out)
	}
	// The 11th and 12th children must NOT appear.
	if strings.Contains(out, "(pid 210)") || strings.Contains(out, "(pid 211)") {
		t.Errorf("PrintTree should omit children beyond the limit; output:\n%s", out)
	}
}

func TestPrintTreeNoChildrenSection(t *testing.T) {
	t.Parallel()

	chain := []model.Process{{PID: 1, Command: "init"}}

	var buf bytes.Buffer
	PrintTree(&buf, chain, nil, false)
	out := buf.String()

	// With no children, the output should be just the ancestry line.
	if strings.Contains(out, "├─") || strings.Contains(out, "and") {
		t.Errorf("PrintTree with no children should not render branch lines; got:\n%s", out)
	}
}

func TestPrintTreeLastChildUsesElbowConnector(t *testing.T) {
	t.Parallel()

	chain := []model.Process{{PID: 100, Command: "nginx"}}
	children := []model.Process{
		{PID: 201, Command: "worker-a"},
		{PID: 202, Command: "worker-b"},
	}

	var buf bytes.Buffer
	PrintTree(&buf, chain, children, false)
	out := buf.String()

	// First child uses ├─, last uses └─. Locate by PID-suffixed substring.
	first := strings.Index(out, "├─ worker-a")
	last := strings.Index(out, "└─ worker-b")
	if first == -1 {
		t.Errorf("expected first child rendered with ├─; output:\n%s", out)
	}
	if last == -1 {
		t.Errorf("expected last child rendered with └─; output:\n%s", out)
	}
}
