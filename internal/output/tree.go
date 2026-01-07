package output

import (
	"fmt"

	"github.com/pranshuparmar/witr/pkg/model"
)

var (
	colorResetTree   = "\033[0m"
	colorMagentaTree = "\033[35m"
	colorGreenTree   = "\033[32m"
	colorBoldTree    = "\033[2m"
)

func PrintTree(chain []model.Process, children []model.Process, colorEnabled bool) {
	colorReset := ""
	colorMagenta := ""
	colorGreen := ""
	colorBold := ""
	if colorEnabled {
		colorReset = colorResetTree
		colorMagenta = colorMagentaTree
		colorGreen = colorGreenTree
		colorBold = colorBoldTree
	}
	for i, p := range chain {
		prefix := ""
		for j := 0; j < i; j++ {
			prefix += "  "
		}
		if i > 0 {
			if colorEnabled {
				prefix += colorMagenta + "└─ " + colorReset
			} else {
				prefix += "└─ "
			}
		}
		if colorEnabled {
			// Highlight target process (last in chain) in green
			cmdColor := ""
			if i == len(chain)-1 {
				cmdColor = colorGreen
			}
			fmt.Printf("%s%s%s%s (%spid %d%s)\n", prefix, cmdColor, p.Command, colorReset, colorBold, p.PID, colorReset)
		} else {
			fmt.Printf("%s%s (pid %d)\n", prefix, p.Command, p.PID)
		}
	}

	// Print children if any
	if len(children) > 0 {
		basePrefix := ""
		for j := 0; j < len(chain); j++ {
			basePrefix += "  "
		}

		limit := 10
		count := len(children)
		for i, child := range children {
			if i >= limit {
				remaining := count - limit
				prefix := basePrefix
				if colorEnabled {
					prefix += colorMagenta + "└─ " + colorReset
				} else {
					prefix += "└─ "
				}
				fmt.Printf("%s... and %d more\n", prefix, remaining)
				break
			}

			connector := "├─ "
			if i == count-1 || (i == limit-1 && count <= limit) {
				connector = "└─ "
			}

			prefix := basePrefix
			if colorEnabled {
				prefix += colorMagenta + connector + colorReset
			} else {
				prefix += connector
			}

			if colorEnabled {
				fmt.Printf("%s%s (%spid %d%s)\n", prefix, child.Command, colorBold, child.PID, colorReset)
			} else {
				fmt.Printf("%s%s (pid %d)\n", prefix, child.Command, child.PID)
			}
		}
	}
}
