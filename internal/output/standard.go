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
	fmt.Printf("\nTarget      : %s\n\n", target)

	// Process
	var proc = r.Ancestry[len(r.Ancestry)-1]
	fmt.Printf("Process     : %s (pid %d)\n", proc.Command, proc.PID)

	// Container
	if proc.Container != "" {
		fmt.Printf("Container   : %s\n", proc.Container)
	}
	// Service
	if proc.Service != "" {
		fmt.Printf("Service     : %s\n", proc.Service)
	}

	if proc.Cmdline != "" {
		fmt.Printf("Command     : %s\n", proc.Cmdline)
	} else {
		fmt.Printf("Command     : %s\n", proc.Command)
	}
	// Format as: 2 days ago (Mon 2025-02-02 11:42:10 +0530)
	startedAt := proc.StartedAt
	now := time.Now()
	dur := now.Sub(startedAt)
	var rel string
	switch {
	case dur.Hours() >= 48:
		days := int(dur.Hours()) / 24
		rel = fmt.Sprintf("%d days ago", days)
	case dur.Hours() >= 24:
		rel = "1 day ago"
	case dur.Hours() >= 2:
		hours := int(dur.Hours())
		rel = fmt.Sprintf("%d hours ago", hours)
	case dur.Minutes() >= 60:
		rel = "1 hour ago"
	default:
		mins := int(dur.Minutes())
		if mins > 0 {
			rel = fmt.Sprintf("%d min ago", mins)
		} else {
			rel = "just now"
		}
	}
	dtStr := startedAt.Format("Mon 2006-01-02 15:04:05 -07:00")
	fmt.Printf("Started     : %s (%s)\n\n", rel, dtStr)

	// Why It Exists (short chain)
	fmt.Printf("Why It Exists :\n  ")
	for i, p := range r.Ancestry {
		name := p.Command
		if name == "" && p.Cmdline != "" {
			name = p.Cmdline
		}
		fmt.Printf("%s (pid %d)", name, p.PID)
		if i < len(r.Ancestry)-1 {
			fmt.Printf(" \u2192 ") // Unicode right arrow
		}
	}
	fmt.Print("\n\n")

	// Source
	sourceLabel := string(r.Source.Type)
	if r.Source.Name != "" && r.Source.Name != sourceLabel {
		fmt.Printf("Source      : %s (%s)\n", r.Source.Name, sourceLabel)
	} else {
		fmt.Printf("Source      : %s\n", sourceLabel)
	}

	// Working Dir
	if proc.WorkingDir != "" {
		fmt.Printf("\nWorking Dir : %s\n", proc.WorkingDir)
	}
	// Git repo/branch
	if proc.GitRepo != "" {
		if proc.GitBranch != "" {
			fmt.Printf("Git Repo    : %s (%s)\n", proc.GitRepo, proc.GitBranch)
		} else {
			fmt.Printf("Git Repo    : %s\n", proc.GitRepo)
		}
	}

	// Listening section (address:port)
	if len(proc.ListeningPorts) > 0 && len(proc.BindAddresses) == len(proc.ListeningPorts) {
		for i := range proc.ListeningPorts {
			addr := proc.BindAddresses[i]
			port := proc.ListeningPorts[i]
			if addr != "" && port > 0 {
				if i == 0 {
					fmt.Printf("Listening   : %s:%d\n", addr, port)
				} else {
					fmt.Printf("              %s:%d\n", addr, port)
				}
			}
		}
	}

	// Warnings
	if len(r.Warnings) > 0 {
		fmt.Println("\nNotes       :")
		for _, w := range r.Warnings {
			fmt.Printf("  • %s\n", w)
		}
	}
	fmt.Println()
}
