package tui

import (
	"testing"

	"github.com/pranshuparmar/witr/pkg/model"
)

func TestGetColumnAtX(t *testing.T) {
	m := InitialModel("test")
	cols := m.table.Columns() // PID(8) is the first column, so col 0 spans [0,10)

	tests := []struct {
		x    int
		want int
	}{
		{0, 0},        // start of PID
		{9, 0},        // last cell of PID (8 + 2 padding)
		{10, 1},       // start of User
		{1 << 20, -1}, // far past the last column
	}
	for _, tt := range tests {
		if got := m.getColumnAtX(tt.x, cols); got != tt.want {
			t.Errorf("getColumnAtX(%d) = %d, want %d", tt.x, got, tt.want)
		}
	}
}

func TestHandleProcessHeaderClick(t *testing.T) {
	m := InitialModel("test")
	m.processes = []model.Process{{PID: 2, MemoryRSS: 1}, {PID: 1, MemoryRSS: 2}}

	// Clicking the PID header (column 0) switches the sort key, defaulting to
	// descending.
	m.handleProcessHeaderClick(0)
	if m.sortCol != "pid" || !m.sortDesc {
		t.Fatalf("first click: sortCol=%q desc=%v, want pid/true", m.sortCol, m.sortDesc)
	}

	// Clicking the same header again just flips the direction.
	m.handleProcessHeaderClick(0)
	if m.sortCol != "pid" || m.sortDesc {
		t.Errorf("second click: sortCol=%q desc=%v, want pid/false", m.sortCol, m.sortDesc)
	}
}

func TestHandlePortHeaderClick(t *testing.T) {
	m := InitialModel("test")
	m.ports = []model.OpenPort{{Port: 80, State: "LISTEN"}}

	// Port table: Port(6) then Protocol(10); column 1 (proto) starts at x=8.
	m.handlePortHeaderClick(8)
	if m.sortPortCol != "proto" || m.sortPortDesc {
		t.Fatalf("port header click: col=%q desc=%v, want proto/false", m.sortPortCol, m.sortPortDesc)
	}
	m.handlePortHeaderClick(8)
	if !m.sortPortDesc {
		t.Errorf("port header re-click: desc=%v, want true", m.sortPortDesc)
	}
}

func TestHandleContainerHeaderClick(t *testing.T) {
	m := InitialModel("test")
	m.containers = []*model.ContainerMatch{{Name: "a"}}

	// Container table: ID is column 0 (x=0).
	m.handleContainerHeaderClick(0)
	if m.sortContainerCol != "id" || m.sortContainerDesc {
		t.Errorf("container header click: col=%q desc=%v, want id/false", m.sortContainerCol, m.sortContainerDesc)
	}
}

func TestHandleLockHeaderClick(t *testing.T) {
	m := InitialModel("test")
	m.locks = []*model.LockedFile{{PID: 1}}

	// Lock table: PID(8) then Process(18); column 1 (process) starts at x=10.
	m.handleLockHeaderClick(10)
	if m.sortLockCol != "process" || m.sortLockDesc {
		t.Errorf("lock header click: col=%q desc=%v, want process/false", m.sortLockCol, m.sortLockDesc)
	}
}
