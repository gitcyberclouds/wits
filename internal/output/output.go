package output

import "github.com/pranshuparmar/witr/pkg/model"

type Renderer interface {
	Render(r Result)
}

type Result struct {
	TargetPID int
	Ancestry  []model.Process
	Source    model.Source
	Warnings  []string
}
