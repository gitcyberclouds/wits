package model

type TargetType string

const (
	TargetName      TargetType = "name"
	TargetPID       TargetType = "pid"
	TargetPort      TargetType = "port"
	TargetFile      TargetType = "file"
	TargetContainer TargetType = "container"
)

type Target struct {
	Type  TargetType
	Value string
}
