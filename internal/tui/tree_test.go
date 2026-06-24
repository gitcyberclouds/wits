package tui

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/pranshuparmar/witr/pkg/model"
)

func TestInitReturnsStartupCommands(t *testing.T) {
	if InitialModel("test").Init() == nil {
		t.Error("Init should return the initial command batch")
	}
}

func TestUpdateContainerDetailMessage(t *testing.T) {
	m := InitialModel("test")
	m, _ = step(t, m, &model.ContainerMatch{Name: "web", ID: "abc123"})
	if m.selectedContainer == nil || m.selectedContainer.Name != "web" {
		t.Errorf("a *ContainerMatch message should populate selectedContainer, got %v", m.selectedContainer)
	}
}

func TestTreeMessagePopulatesViewport(t *testing.T) {
	m := InitialModel("test")
	m.processes = []model.Process{{PID: 42, Command: "x"}}
	m.filterProcesses() // the selected row now carries PID 42

	res := model.Result{
		Process:  model.Process{PID: 42, Command: "x", Cmdline: "x --flag"},
		Ancestry: []model.Process{{PID: 1, Command: "systemd"}, {PID: 42, Command: "x"}},
		Children: []model.Process{{PID: 100, Command: "child"}},
	}
	m, _ = step(t, m, treeMsg(res))

	// handleTree -> updateTreeViewport -> renderTreeContent: ancestry then child.
	if want := []int{1, 42, 100}; !equalInts(m.treePIDs, want) {
		t.Fatalf("treePIDs = %v, want %v", m.treePIDs, want)
	}

	// Navigating the tree in the side pane re-renders it (rerenderTree).
	m.listFocus = focusSide
	before := m.treeCursor
	m, _ = step(t, m, tea.KeyMsg{Type: tea.KeyUp})
	if m.treeCursor == before {
		t.Errorf("up in the side pane should move the tree cursor from %d", before)
	}
}

func TestDebounceMessageFetchesSelection(t *testing.T) {
	m := InitialModel("test")
	m.processes = []model.Process{{PID: 42, Command: "x"}}
	m.filterProcesses()
	m.selectionID = 7 // the debounce must carry the live selection id to fire

	_, cmd := step(t, m, debounceMsg{id: 7, pid: 42})
	if cmd == nil {
		t.Error("a debounce matching the current selection should fetch the tree")
	}
}
