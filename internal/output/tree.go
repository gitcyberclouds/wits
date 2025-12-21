package output

import (
	"fmt"

	"github.com/pranshuparmar/witr/pkg/model"
)

func PrintTree(chain []model.Process) {
	for i, p := range chain {
		prefix := ""
		for j := 0; j < i; j++ {
			prefix += "  "
		}
		if i > 0 {
			prefix += "└─ "
		}
		fmt.Printf("%s%s (pid %d)\n", prefix, p.Command, p.PID)
	}
}
