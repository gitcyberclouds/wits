package proc

import (
	"sort"

	"github.com/pranshuparmar/witr/pkg/model"
)

func sortProcesses(processes []model.Process) {
	sort.Slice(processes, func(i, j int) bool {
		return processes[i].PID < processes[j].PID
	})
}
