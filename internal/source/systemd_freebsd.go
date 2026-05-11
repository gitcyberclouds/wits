//go:build freebsd

package source

import "github.com/pranshuparmar/witr/pkg/model"

func detectSystemd(_ []model.Process) *model.Source {
	// FreeBSD doesn't use systemd
	return nil
}

// IsSystemdRunning always returns false on FreeBSD.
func IsSystemdRunning() bool { return false }
