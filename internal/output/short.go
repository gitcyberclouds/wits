package output

import (
	"io"

	"github.com/pranshuparmar/witr/pkg/model"
)

var (
	colorResetShort   = ansiString("\033[0m")
	colorMagentaShort = ansiString("\033[35m")
	colorBoldShort    = ansiString("\033[2m")
)

func RenderShort(w io.Writer, r model.Result, colorEnabled bool) {
	p := NewPrinter(w)

	for i, proc := range r.Ancestry {
		if i > 0 {
			if colorEnabled {
				p.Printf("%s → %s", colorMagentaShort, colorResetShort)
			} else {
				p.Print(" → ")
			}
		}
		if colorEnabled {
			p.Printf("%s (%spid %d%s)", proc.Command, colorBoldShort, proc.PID, colorResetShort)
		} else {
			p.Printf("%s (pid %d)", proc.Command, proc.PID)
		}
	}
	p.Println()
}
