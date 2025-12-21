package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/pranshuparmar/witr/internal/process"
	"github.com/pranshuparmar/witr/internal/source"
	"github.com/pranshuparmar/witr/internal/target"
	"github.com/pranshuparmar/witr/pkg/model"
)

func main() {
	pidFlag := flag.String("pid", "", "pid to explain")
	portFlag := flag.String("port", "", "port to explain")
	shortFlag := flag.Bool("short", false, "short output")
	treeFlag := flag.Bool("tree", false, "tree output")

	flag.Parse()

	var t model.Target

	switch {
	case *pidFlag != "":
		t = model.Target{Type: model.TargetPID, Value: *pidFlag}
	case *portFlag != "":
		t = model.Target{Type: model.TargetPort, Value: *portFlag}
	case flag.NArg() == 1:
		t = model.Target{Type: model.TargetName, Value: flag.Arg(0)}
	default:
		fmt.Println("usage: witr [--pid N | --port N | name]")
		os.Exit(1)
	}

	pids, err := target.Resolve(t)
	if err != nil {
		fmt.Println("error:", err)
		os.Exit(1)
	}

	if len(pids) > 1 {
		fmt.Println("Multiple matching processes found:\n")
		for _, pid := range pids {
			fmt.Printf("  PID %d\n", pid)
		}
		fmt.Println("\nRe-run with:")
		fmt.Println("  witr --pid <pid>")
		os.Exit(1)
	}

	pid := pids[0]

	ancestry, err := process.BuildAncestry(pid)
	if err != nil {
		fmt.Println("error:", err)
		os.Exit(1)
	}

	src := source.Detect(ancestry)

	// TEMP output (we’ll clean this next)
	fmt.Println("Why It Exists:")
	for i, p := range ancestry {
		if i > 0 {
			fmt.Print(" → ")
		}
		fmt.Print(p.Command)
	}

	fmt.Printf("\n\nSource: %s\n", src.Type)

	_ = shortFlag
	_ = treeFlag
}
