package model

// LockedFile describes a file lock held by a process.
type LockedFile struct {
	PID     int
	Process string
	Path    string
	Type    string // POSIX, FLOCK, OFDLCK
	Mode    string // READ, WRITE, RW
}
