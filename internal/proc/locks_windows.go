//go:build windows

package proc

import "github.com/pranshuparmar/witr/pkg/model"

// ListLockedFiles returns nil on Windows. Windows uses file sharing modes
// rather than POSIX-style advisory locks, and there's no public API for
// enumerating all current locks system-wide.
func ListLockedFiles() []*model.LockedFile { return nil }
