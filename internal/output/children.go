package output

import (
	"fmt"

	"github.com/pranshuparmar/witr/pkg/model"
)

func PrintChildren(root model.Process, children []model.Process, colorEnabled bool) {
	rootLine := formatProcessLine(root, colorEnabled)
	if colorEnabled {
		fmt.Printf("%sChildren%s of %s:\n", colorMagentaTree, colorResetTree, rootLine)
	} else {
		fmt.Printf("Children of %s:\n", rootLine)
	}

	if len(children) == 0 {
		if colorEnabled {
			fmt.Printf("%sNo child processes found.%s\n", colorGreen, colorReset)
		} else {
			fmt.Println("No child processes found.")
		}
		return
	}

	for i, child := range children {
		isLast := i == len(children)-1
		prefix := treeConnector(isLast, colorEnabled)
		fmt.Printf("  %s%s\n", prefix, formatProcessLine(child, colorEnabled))
	}
}

func treeConnector(isLast bool, colorEnabled bool) string {
	connector := "├─ "
	if isLast {
		connector = "└─ "
	}
	if colorEnabled {
		return colorMagentaTree + connector + colorResetTree
	}
	return connector
}

func formatProcessLine(proc model.Process, colorEnabled bool) string {
	name := proc.Command
	if name == "" && proc.Cmdline != "" {
		name = proc.Cmdline
	}
	if name == "" {
		name = "unknown"
	}
	if colorEnabled {
		return fmt.Sprintf("%s (%spid %d%s)", name, colorBoldTree, proc.PID, colorResetTree)
	}
	return fmt.Sprintf("%s (pid %d)", name, proc.PID)
}
