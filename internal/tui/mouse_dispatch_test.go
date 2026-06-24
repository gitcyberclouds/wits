package tui

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/pranshuparmar/witr/pkg/model"
)

func click(x, y int) tea.MouseMsg {
	return tea.MouseMsg{Action: tea.MouseActionPress, Button: tea.MouseButtonLeft, X: x, Y: y}
}

func TestMouseTitleClickGoesHome(t *testing.T) {
	m := InitialModel("test")
	m.activeTab = tabPorts
	m.state = stateDetail

	m, _ = step(t, m, click(3, 1)) // "witr" title sits at x 1..6, y 1
	if m.activeTab != tabProcesses || m.state != stateList {
		t.Errorf("title click should go home: tab=%v state=%v", m.activeTab, m.state)
	}
}

func TestMouseTabClick(t *testing.T) {
	m := InitialModel("test")       // starts on Processes
	m, _ = step(t, m, click(24, 1)) // "2. Ports" header range
	if m.activeTab != tabPorts {
		t.Errorf("clicking the Ports tab should switch to it, got %v", m.activeTab)
	}
}

func TestMouseSearchRowFocusesInput(t *testing.T) {
	m := InitialModel("test")
	m, _ = step(t, m, click(10, 5)) // the search input row is y == 5
	if !m.input.Focused() {
		t.Error("clicking the search row should focus the filter input")
	}
}

func TestMouseProcessHeaderClickSorts(t *testing.T) {
	m := InitialModel("test")
	m, _ = step(t, m, tea.WindowSizeMsg{Width: 120, Height: 40})
	m.processes = []model.Process{{PID: 1}, {PID: 2}}
	m.filterProcesses()

	// Content header row is y == 7; x == 2 maps to content column 0 (PID).
	m, _ = step(t, m, click(2, 7))
	if m.sortCol != "pid" {
		t.Errorf("clicking the PID header should sort by pid, got %q", m.sortCol)
	}
}

func TestMouseProcessWheelScrolls(t *testing.T) {
	m := InitialModel("test")
	m.processes = []model.Process{{PID: 1}, {PID: 2}, {PID: 3}}
	m.filterProcesses()

	wheel := tea.MouseMsg{Action: tea.MouseActionPress, Button: tea.MouseButtonWheelDown, X: 5, Y: 10}
	m, _ = step(t, m, wheel)
	if m.table.Cursor() != 1 {
		t.Errorf("wheel-down in the list should advance the cursor to 1, got %d", m.table.Cursor())
	}
}
