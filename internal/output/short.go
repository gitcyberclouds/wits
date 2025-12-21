package output

import "fmt"

func RenderShort(r Result) {
	for i, p := range r.Ancestry {
		if i > 0 {
			fmt.Print(" → ")
		}
		fmt.Print(p.Command)
	}
	fmt.Println()
	fmt.Println("source:", r.Source.Type)
}
