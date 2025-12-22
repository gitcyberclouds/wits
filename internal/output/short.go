package output

import (
	"fmt"

	"github.com/pranshuparmar/witr/pkg/model"
)

var (
	colorResetShort   = "\033[0m"
	colorMagentaShort = "\033[35m"
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
		fmt.Print(p.Command)
	}
	fmt.Println()
}
