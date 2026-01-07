package output

import (
	"fmt"

	"github.com/pranshuparmar/witr/pkg/model"
)

var (
	colorResetShort   = "\033[0m"
	colorMagentaShort = "\033[35m"
	colorBoldShort    = "\033[2m"
	colorGreenShort   = "\033[32m"
)

func RenderShort(r model.Result, colorEnabled bool) {
	for i, p := range r.Ancestry {
		if i > 0 {
			if colorEnabled {
				fmt.Print(colorMagentaShort + " → " + colorResetShort)
			} else {
				fmt.Print(" → ")
			}
		}

		if colorEnabled {
			nameColor := ""
			if i == len(r.Ancestry)-1 {
				nameColor = colorGreenShort
			}

			fmt.Printf("%s%s%s (%spid %d%s)", nameColor, p.Command, colorResetShort, colorBoldShort, p.PID, colorResetShort)
		} else {
			fmt.Printf("%s (pid %d)", p.Command, p.PID)
		}
	}
	fmt.Println()
}
