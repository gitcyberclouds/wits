package model

import "time"

type Process struct {
	PID        int
	PPID       int
	Command    string
	Executable string
	StartTime  time.Time
}
