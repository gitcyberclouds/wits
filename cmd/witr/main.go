package main

import (
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/pranshuparmar/witr/internal/output"
	"github.com/pranshuparmar/witr/internal/process"
	"github.com/pranshuparmar/witr/internal/source"
	"github.com/pranshuparmar/witr/internal/target"
	"github.com/pranshuparmar/witr/pkg/model"
)

func printHelp() {
	fmt.Println("Usage: witr [--pid N | --port N | name] [--short] [--tree] [--json] [--warnings] [--no-color] [--help]")
	fmt.Println("  --pid <n>         Explain a specific PID")
	fmt.Println("  --port <n>        Explain port usage")
	fmt.Println("  --short           One-line summary")
	fmt.Println("  --tree            Show full process ancestry tree")
	fmt.Println("  --json            Output result as JSON")
	fmt.Println("  --warnings        Show only warnings")
	fmt.Println("  --no-color        Disable colorized output")
	fmt.Println("  --help            Show this help message")
}

func main() {
	pidFlag := flag.String("pid", "", "pid to explain")
	portFlag := flag.String("port", "", "port to explain")
	shortFlag := flag.Bool("short", false, "short output")
	treeFlag := flag.Bool("tree", false, "tree output")
	jsonFlag := flag.Bool("json", false, "output as JSON")
	warnFlag := flag.Bool("warnings", false, "show only warnings")
	noColorFlag := flag.Bool("no-color", false, "disable colorized output")
	helpFlag := flag.Bool("help", false, "show help")

	flag.Parse()

	if *helpFlag {
		printHelp()
		os.Exit(0)
	}

	var t model.Target

	switch {
	case *pidFlag != "":
		t = model.Target{Type: model.TargetPID, Value: *pidFlag}
	case *portFlag != "":
		t = model.Target{Type: model.TargetPort, Value: *portFlag}
	case flag.NArg() == 1:
		t = model.Target{Type: model.TargetName, Value: flag.Arg(0)}
	default:
		printHelp()
		os.Exit(1)
	}

	pids, err := target.Resolve(t)
	if err != nil {
		errStr := err.Error()
		fmt.Println()
		fmt.Println("Error:")
		fmt.Printf("  %s\n", errStr)
		if strings.Contains(errStr, "socket found but owning process not detected") {
			fmt.Println("\nA socket was found for the port, but the owning process could not be detected.")
			fmt.Println("This may be due to insufficient permissions. Try running with sudo:")
			// Print the actual command the user entered, prefixed with sudo
			fmt.Print("  sudo ")
			for i, arg := range os.Args {
				if i > 0 {
					fmt.Print(" ")
				}
				fmt.Print(arg)
			}
			fmt.Println()
		} else {
			fmt.Println("\nNo matching process or service found. Please check your query or try a different name/port/PID.")
		}
		fmt.Println("For usage and options, run: witr --help")
		os.Exit(1)
	}

	if len(pids) > 1 {
		fmt.Print("Multiple matching processes found:\n\n")
		for i, pid := range pids {
			cmdline := "(unknown)"
			cmdlineBytes, err := os.ReadFile(fmt.Sprintf("/proc/%d/cmdline", pid))
			if err == nil {
				cmd := strings.ReplaceAll(string(cmdlineBytes), "\x00", " ")
				cmdline = strings.TrimSpace(cmd)
			}
			fmt.Printf("[%d] PID %d   %s\n", i+1, pid, cmdline)
		}
		fmt.Println("\nRe-run with:")
		fmt.Println("  witr --pid <pid>")
		os.Exit(1)
	}

	pid := pids[0]

	ancestry, err := process.BuildAncestry(pid)
	if err != nil {
		fmt.Println()
		fmt.Println("Error:")
		fmt.Printf("  %s\n", err.Error())
		fmt.Println("\nNo matching process or service found. Please check your query or try a different name/port/PID.")
		fmt.Println("For usage and options, run: witr --help")
		os.Exit(1)
	}

	src := source.Detect(ancestry)

	var proc model.Process
	if len(ancestry) > 0 {
		proc = ancestry[len(ancestry)-1]
	}
	resolvedTarget := "unknown"
	if len(ancestry) > 0 {
		proc = ancestry[len(ancestry)-1]
		resolvedTarget = proc.Command
	}

	res := model.Result{
		Target:         t,
		ResolvedTarget: resolvedTarget,
		Process:        proc,
		Ancestry:       ancestry,
		Source:         src,
		Warnings:       source.Warnings(ancestry),
	}

	switch {
	case *jsonFlag:
		// Output as JSON
		importJson, _ := output.ToJSON(res)
		fmt.Println(importJson)
	case *warnFlag:
		if len(res.Warnings) == 0 {
			fmt.Println("No warnings.")
		} else {
			fmt.Println("Warnings:")
			for _, w := range res.Warnings {
				fmt.Printf("  • %s\n", w)
			}
		}
	case *treeFlag:
		output.PrintTree(res.Ancestry) // (color support can be added here if needed)
	case *shortFlag:
		output.RenderShort(res, !*noColorFlag)
	default:
		output.RenderStandard(res, !*noColorFlag)
	}

	_ = shortFlag
	_ = treeFlag
}
