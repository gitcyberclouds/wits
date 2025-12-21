package output

import (
	"fmt"
	"time"

	"github.com/pranshuparmar/witr/pkg/model"
)

func RenderStandard(r model.Result) {
	fmt.Println("Why is this running?")
	fmt.Println()

	for i, p := range r.Ancestry {
		prefix := "└─"
		if i < len(r.Ancestry)-1 {
			prefix = "├─"
		}

		started := p.StartedAt.Format("2006-01-02 15:04:05")
		ago := time.Since(p.StartedAt).Round(time.Second)

		fmt.Printf(
			"%s %s (pid=%d)\n   started: %s (%s ago)\n",
			prefix,
			p.Command,
			p.PID,
			started,
			ago,
		)
	}

	fmt.Println()
	fmt.Printf("Source: %s\n", r.Source.Type)

	if len(r.Warnings) > 0 {
		fmt.Println()
		fmt.Println("Warnings:")
		for _, w := range r.Warnings {
			fmt.Println(" -", w)
		}
	}
}
