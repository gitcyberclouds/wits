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

func PrintDescendants(tree model.ProcessTree, colorEnabled bool) {
	fmt.Printf("%s\n", formatProcessLine(tree.Process, colorEnabled))
	for i, child := range tree.Children {
		isLast := i == len(tree.Children)-1
		printTreeNode(child, "", isLast, colorEnabled)
	}
}

func printTreeNode(node model.ProcessTree, prefix string, isLast bool, colorEnabled bool) {
	connector := treeConnector(isLast, colorEnabled)
	fmt.Printf("%s%s%s\n", prefix, connector, formatProcessLine(node.Process, colorEnabled))

	nextPrefix := prefix
	if isLast {
		nextPrefix += "   "
	} else {
		nextPrefix += "│  "
	}

	for i, child := range node.Children {
		childLast := i == len(node.Children)-1
		printTreeNode(child, nextPrefix, childLast, colorEnabled)
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
