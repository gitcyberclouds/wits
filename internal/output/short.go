package output

import (
	"fmt"

	"github.com/pranshuparmar/witr/pkg/model"
)

func RenderShort(r model.Result) {
	for i, p := range r.Ancestry {
		if i > 0 {
			fmt.Print(" → ")
		}
		fmt.Print(p.Command)
	}
	fmt.Println()
}
