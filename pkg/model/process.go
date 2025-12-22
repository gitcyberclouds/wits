package model

import "time"

type Process struct {
	PID       int
	PPID      int
	Command   string
	Cmdline   string
	Exe       string
	StartedAt time.Time
	User      string

	WorkingDir string
	GitRepo    string
	GitBranch  string
	Container  string
	Service    string

	// Network context
	ListeningPorts []int
	BindAddresses  []string
}
