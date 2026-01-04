package output

import (
	"io"
	"strings"

	"github.com/pranshuparmar/witr/pkg/model"
)

var (
	colorResetTree   = ansiString("\033[0m")
	colorMagentaTree = ansiString("\033[35m")
	colorBoldTree    = ansiString("\033[2m")
)

func PrintTree(w io.Writer, chain []model.Process, colorEnabled bool) {
	p := NewPrinter(w)

	for i, proc := range chain {
		indent := strings.Repeat("  ", i)
		if i > 0 {
			if colorEnabled {
				p.Printf("%s%s└─ %s", indent, colorMagentaTree, colorResetTree)
			} else {
				p.Printf("%s└─ ", indent)
			}
		}
		if colorEnabled {
			p.Printf("%s (%spid %d%s)\n", proc.Command, colorBoldTree, proc.PID, colorResetTree)
		} else {
			p.Printf("%s (pid %d)\n", proc.Command, proc.PID)
		}
	}
}
