package output

import (
	"fmt"

	"github.com/pranshuparmar/witr/pkg/model"
)

var (
	colorResetTree   = "\033[0m"
	colorMagentaTree = "\033[35m"
)

func PrintTree(chain []model.Process) {
	for i, p := range chain {
		prefix := ""
		for j := 0; j < i; j++ {
			prefix += "  "
		}
		if i > 0 {
			if true { // always colorize separator if color is enabled (could add flag if needed)
				prefix += colorMagentaTree + "└─ " + colorResetTree
			} else {
				prefix += "└─ "
			}
		}
		fmt.Printf("%s%s (pid %d)\n", prefix, p.Command, p.PID)
	}
}
