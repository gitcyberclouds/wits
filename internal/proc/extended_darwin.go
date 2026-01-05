//go:build darwin

package proc

import "github.com/pranshuparmar/witr/pkg/model"

// ReadExtendedInfo is a no-op on macOS; verbose extended info is Linux-only.
func ReadExtendedInfo(pid int) (model.MemoryInfo, model.IOStats, []string, int, uint64, []int, int, error) {
	return model.MemoryInfo{}, model.IOStats{}, nil, 0, 0, nil, 0, nil
}
