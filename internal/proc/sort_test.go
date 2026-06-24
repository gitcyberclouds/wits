package proc

import (
	"testing"

	"github.com/pranshuparmar/witr/pkg/model"
)

func TestSortProcesses(t *testing.T) {
	ps := []model.Process{{PID: 30}, {PID: 10}, {PID: 20}}
	sortProcesses(ps)
	for i, want := range []int{10, 20, 30} {
		if ps[i].PID != want {
			t.Errorf("after sort, ps[%d].PID = %d, want %d", i, ps[i].PID, want)
		}
	}
}
