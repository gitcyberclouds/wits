package model

import "time"

type Process struct {
	PID       int
	PPID      int
	Command   string
	Exe       string
	StartedAt time.Time
	User      string

	WorkingDir string

	// Network context
	ListeningPorts []int
	BindAddresses  []string
}
