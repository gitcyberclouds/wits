package output

import (
	"strings"

	"github.com/pranshuparmar/witr/pkg/model"
)

func FormatWhy(chain []model.Process) string {
	var names []string
	for _, p := range chain {
		names = append(names, p.Command)
	}
	return strings.Join(names, " → ")
}
