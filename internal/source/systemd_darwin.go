//go:build darwin

package source

import "github.com/pranshuparmar/witr/pkg/model"

func detectSystemd(_ []model.Process) *model.Source {
	return nil
}

// IsSystemdRunning always returns false on macOS.
func IsSystemdRunning() bool { return false }
