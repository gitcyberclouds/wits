package output

import (
	"bytes"
	"fmt"
	"strings"
	"testing"

	"github.com/pranshuparmar/witr/pkg/model"
)

func TestPrintChildren(t *testing.T) {
	root := model.Process{PID: 1, Command: "parent"}

	t.Run("no children", func(t *testing.T) {
		for _, color := range []bool{false, true} {
			var buf bytes.Buffer
			PrintChildren(&buf, root, nil, color)
			if !strings.Contains(buf.String(), "No child processes found.") {
				t.Errorf("color=%v: missing empty-state message", color)
			}
		}
	})

	t.Run("with children", func(t *testing.T) {
		kids := []model.Process{{PID: 2, Command: "a"}, {PID: 3, Command: "b"}}
		var plain bytes.Buffer
		PrintChildren(&plain, root, kids, false)
		if !strings.Contains(plain.String(), "Children of parent") || !strings.Contains(plain.String(), "(pid 2)") {
			t.Errorf("plain children render wrong:\n%s", plain.String())
		}
		var colored bytes.Buffer
		PrintChildren(&colored, root, kids, true)
		if !strings.Contains(colored.String(), "parent") || !strings.Contains(colored.String(), "pid 3") {
			t.Errorf("colored children render wrong:\n%s", colored.String())
		}
	})

	t.Run("truncates beyond 10", func(t *testing.T) {
		many := make([]model.Process, 15)
		for i := range many {
			many[i] = model.Process{PID: i + 10, Command: fmt.Sprintf("c%d", i)}
		}
		var buf bytes.Buffer
		PrintChildren(&buf, root, many, false)
		if !strings.Contains(buf.String(), "and 5 more") {
			t.Errorf("expected truncation note; got:\n%s", buf.String())
		}
	})

	t.Run("falls back to unknown name", func(t *testing.T) {
		var buf bytes.Buffer
		PrintChildren(&buf, model.Process{PID: 9}, []model.Process{{PID: 10}}, false)
		if !strings.Contains(buf.String(), "unknown") {
			t.Errorf("expected unknown-name fallback; got:\n%s", buf.String())
		}
	})
}
