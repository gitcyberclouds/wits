package main

import (
	"fmt"
	"os"
	"strconv"

	"github.com/pranshuparmar/witr/internal/process"
	"github.com/pranshuparmar/witr/internal/source"
	"github.com/pranshuparmar/witr/internal/target"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("usage: witr <pid>")
		return
	}

	if os.Args[1] == "--port" {
		port, _ := strconv.Atoi(os.Args[2])
		pids, err := target.ResolvePort(port)
		if err != nil {
			fmt.Println("error:", err)
			return
		}
		fmt.Println("PIDs listening:", pids)
		return
	}

	pid, _ := strconv.Atoi(os.Args[1])

	ancestry, err := process.BuildAncestry(pid)
	if err != nil {
		fmt.Println("error:", err)
		return
	}

	src := source.Detect(ancestry)

	fmt.Println("Why It Exists:")
	for i, p := range ancestry {
		if i > 0 {
			fmt.Print(" → ")
		}
		fmt.Print(p.Command)
	}

	fmt.Println("\n\nSource:")
	fmt.Printf("  %s (confidence %.1f)\n", src.Type, src.Confidence)
}
