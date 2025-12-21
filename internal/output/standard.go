package output

import (
	"fmt"
	"time"

	"github.com/pranshuparmar/witr/pkg/model"
)

func RenderStandard(r model.Result) {
	// Target
	target := "unknown"
	if len(r.Ancestry) > 0 {
		target = r.Ancestry[len(r.Ancestry)-1].Command
	}
	fmt.Printf("Target      : %s\n\n", target)

	// Process
	proc := r.Ancestry[len(r.Ancestry)-1]
	fmt.Printf("Process     : %s (pid %d)\n", proc.Command, proc.PID)
	fmt.Printf("Command     : %s\n", proc.Command)
	fmt.Printf("Started     : %s (%s ago)\n\n",
		proc.StartedAt.Format("2006-01-02 15:04:05"),
		time.Since(proc.StartedAt).Round(time.Second))

	// Why It Exists
	fmt.Printf("Why It Exists :\n  ")
	for i, p := range r.Ancestry {
		if i > 0 {
			fmt.Print(" → ")
		}
		fmt.Print(p.Command)
	}
	fmt.Println("\n")

	// Source
	fmt.Printf("Source      : %s\n", r.Source.Type)

	// Working Dir
	if proc.WorkingDir != "" {
		fmt.Printf("Working Dir : %s\n", proc.WorkingDir)
	}

	// Warnings
	if len(r.Warnings) > 0 {
		fmt.Println("\nNotes       :")
		for _, w := range r.Warnings {
			fmt.Printf("  • %s\n", w)
		}
	}
}
