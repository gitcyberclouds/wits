package tui

import (
	"strings"
	"testing"
	"time"

	"github.com/pranshuparmar/witr/pkg/model"
)

// pids extracts the PID order from a process slice for concise assertions.
func pids(ps []model.Process) []int {
	out := make([]int, len(ps))
	for i, p := range ps {
		out[i] = p.PID
	}
	return out
}

func TestFilterProcesses(t *testing.T) {
	m := InitialModel("test")
	m.processes = []model.Process{
		{PID: 1, Command: "nginx", User: "root"},
		{PID: 2, Command: "redis", User: "redis"},
		{PID: 3, Command: "nginx-worker", User: "www"},
	}

	// Empty filter keeps everything and mirrors into the table rows.
	m.input.SetValue("")
	m.filterProcesses()
	if len(m.filtered) != 3 {
		t.Fatalf("empty filter: filtered = %d, want 3", len(m.filtered))
	}
	if got := len(m.table.Rows()); got != 3 {
		t.Errorf("empty filter: table rows = %d, want 3 (filtered must drive the table)", got)
	}

	// Substring filter narrows to matching commands, preserving order.
	m.input.SetValue("nginx")
	m.filterProcesses()
	if got := pids(m.filtered); !equalInts(got, []int{1, 3}) {
		t.Errorf("nginx filter: %v, want [1 3]", got)
	}

	// A filter that matches nothing clears the list.
	m.input.SetValue("zzz")
	m.filterProcesses()
	if len(m.filtered) != 0 {
		t.Errorf("no-match filter: filtered = %d, want 0", len(m.filtered))
	}
}

func TestSortProcessesModel(t *testing.T) {
	base := []model.Process{
		{PID: 30, Command: "Bravo", User: "u", CPUPercent: 1, MemoryRSS: 100, StartedAt: time.Unix(300, 0)},
		{PID: 10, Command: "alpha", User: "u", CPUPercent: 3, MemoryRSS: 300, StartedAt: time.Unix(100, 0)},
		{PID: 20, Command: "charlie", User: "u", CPUPercent: 2, MemoryRSS: 200, StartedAt: time.Unix(200, 0)},
	}
	load := func() MainModel {
		m := InitialModel("test")
		m.processes = append([]model.Process(nil), base...)
		return m
	}

	tests := []struct {
		name string
		col  string
		desc bool
		want []int
	}{
		{"pid asc", "pid", false, []int{10, 20, 30}},
		{"mem desc", "mem", true, []int{10, 20, 30}},                      // RSS 300,200,100
		{"cpu asc", "cpu", false, []int{30, 20, 10}},                      // 1,2,3
		{"name asc (case-insensitive)", "name", false, []int{10, 30, 20}}, // alpha,bravo,charlie
		{"time asc", "time", false, []int{10, 20, 30}},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			m := load()
			m.sortCol, m.sortDesc = tt.col, tt.desc
			m.sortProcesses()
			if got := pids(m.processes); !equalInts(got, tt.want) {
				t.Errorf("sort %s = %v, want %v", tt.name, got, tt.want)
			}
		})
	}
}

func TestSortPortsModel(t *testing.T) {
	m := InitialModel("test")
	m.ports = []model.OpenPort{
		{Port: 443, Protocol: "tcp"},
		{Port: 80, Protocol: "udp"},
		{Port: 22, Protocol: "tcp"},
	}

	m.sortPortCol, m.sortPortDesc = "port", false
	m.sortPorts()
	if got := []int{m.ports[0].Port, m.ports[1].Port, m.ports[2].Port}; !equalInts(got, []int{22, 80, 443}) {
		t.Errorf("port asc = %v, want [22 80 443]", got)
	}

	m.sortPortDesc = true
	m.sortPorts()
	if m.ports[0].Port != 443 {
		t.Errorf("port desc head = %d, want 443", m.ports[0].Port)
	}
}

func TestSortContainersModel(t *testing.T) {
	m := InitialModel("test")
	m.containers = []*model.ContainerMatch{{Name: "web"}, {Name: "Api"}, {Name: "db"}}

	m.sortContainerCol, m.sortContainerDesc = "name", false
	m.sortContainers()
	want := []string{"Api", "db", "web"} // case-insensitive
	for i, w := range want {
		if m.containers[i].Name != w {
			t.Errorf("container[%d] = %q, want %q", i, m.containers[i].Name, w)
		}
	}
}

func TestSortLocksModel(t *testing.T) {
	m := InitialModel("test")
	m.locks = []*model.LockedFile{{PID: 30}, {PID: 10}, {PID: 20}}

	m.sortLockCol, m.sortLockDesc = "pid", false
	m.sortLocks()
	for i, w := range []int{10, 20, 30} {
		if m.locks[i].PID != w {
			t.Errorf("lock[%d].PID = %d, want %d", i, m.locks[i].PID, w)
		}
	}
}

func TestGetColumns(t *testing.T) {
	m := InitialModel("test")
	m.sortCol, m.sortDesc = "pid", true

	cols := m.getColumns()
	if len(cols) != 6 {
		t.Fatalf("default cols = %d, want 6 (Command hidden)", len(cols))
	}
	if !strings.Contains(cols[0].Title, "↓") {
		t.Errorf("PID header should carry a desc arrow, got %q", cols[0].Title)
	}

	m.showCmdCol = true
	cols = m.getColumns()
	if len(cols) != 7 || cols[6].Title != "Command" {
		t.Errorf("showCmdCol cols = %d (last %q), want 7 ending in Command", len(cols), cols[len(cols)-1].Title)
	}
}

func TestGetSecondaryColumnsArrows(t *testing.T) {
	m := InitialModel("test")

	m.sortPortCol, m.sortPortDesc = "state", true
	if got := m.getPortColumns()[3].Title; !strings.Contains(got, "↓") {
		t.Errorf("port State header = %q, want desc arrow", got)
	}

	m.sortContainerCol, m.sortContainerDesc = "image", false
	if got := m.getContainerColumns()[3].Title; !strings.Contains(got, "↑") {
		t.Errorf("container Image header = %q, want asc arrow", got)
	}

	m.sortLockCol, m.sortLockDesc = "path", false
	if got := m.getLockColumns()[4].Title; !strings.Contains(got, "↑") {
		t.Errorf("lock Path header = %q, want asc arrow", got)
	}
}
