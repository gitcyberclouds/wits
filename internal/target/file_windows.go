package target

import (
	"fmt"
)

func ResolveFile(path string) ([]int, error) {
	return nil, fmt.Errorf("finding process by file is not supported on Windows")
}
