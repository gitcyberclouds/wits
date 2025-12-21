package model

type TargetType string

const (
	TargetName TargetType = "name"
	TargetPID  TargetType = "pid"
	TargetPort TargetType = "port"
)

type Target struct {
	Type  TargetType
	Value string
}
