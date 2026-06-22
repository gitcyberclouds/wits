package output

import (
	"io"

	"github.com/pranshuparmar/witr/pkg/model"
)

func RenderShort(w io.Writer, r model.Result, colorEnabled bool) {
	p := NewPrinter(w)

	for i, proc := range r.Ancestry {
		if i > 0 {
			if colorEnabled {
				p.Printf("%s → %s", ColorMagenta, ColorReset)
			} else {
				p.Print(" → ")
			}
		}

		if colorEnabled {
			nameColor := ansiString("")
			if i == len(r.Ancestry)-1 {
				nameColor = ColorGreen
			}
			p.Printf("%s%s%s (%spid %d%s)", nameColor, ChainName(proc), ColorReset, ColorDim, proc.PID, ColorReset)
		} else {
			p.Printf("%s (pid %d)", ChainName(proc), proc.PID)
		}
	}
	p.Println()
}

// ChainName returns a display name for an ancestry or child node, falling back
// to the command line and then a placeholder when the process name couldn't be
// read (e.g. a protected or already-exited Windows ancestor that exposes
// neither an image name nor a command line).
func ChainName(p model.Process) string {
	if p.Command != "" {
		return p.Command
	}
	if p.Cmdline != "" {
		return p.Cmdline
	}
	return "(unknown)"
}
