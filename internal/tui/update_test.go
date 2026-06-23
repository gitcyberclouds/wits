package tui

import (
	"fmt"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/pranshuparmar/witr/pkg/model"
)

// keyRunes builds a rune key message (e.g. "p", "/", "a") the way bubbletea
// delivers ordinary character presses.
func keyRunes(s string) tea.KeyMsg { return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(s)} }

// step runs one Update and returns the concrete model, failing if the model
// type ever changes out from under us.
func step(t *testing.T, m MainModel, msg tea.Msg) (MainModel, tea.Cmd) {
	t.Helper()
	nm, cmd := m.Update(msg)
	mm, ok := nm.(MainModel)
	if !ok {
		t.Fatalf("Update returned %T, want MainModel", nm)
	}
	return mm, cmd
}

func TestUpdateQuitKeys(t *testing.T) {
	for _, key := range []tea.KeyMsg{{Type: tea.KeyCtrlC}, keyRunes("q"), {Type: tea.KeyEsc}} {
		m, cmd := step(t, InitialModel("test"), key)
		if !m.quitting {
			t.Errorf("%s should set quitting", key)
		}
		if cmd == nil {
			t.Errorf("%s should return a quit command", key)
		}
	}
}

func TestUpdateTabSwitch(t *testing.T) {
	tests := []struct {
		key  string
		want tab
	}{
		{"2", tabPorts},
		{"3", tabContainers},
		{"1", tabProcesses},
	}
	for _, tt := range tests {
		// Start on a different tab so the switch is observable.
		m := InitialModel("test")
		m.activeTab = tabLocks
		m, _ = step(t, m, keyRunes(tt.key))
		if m.activeTab != tt.want {
			t.Errorf("key %q: activeTab = %v, want %v", tt.key, m.activeTab, tt.want)
		}
	}
}

func TestUpdateEnterOpensProcessDetail(t *testing.T) {
	m := InitialModel("test")
	m.processes = []model.Process{{PID: 4242, Command: "x"}}
	m.filterProcesses() // gives the table a selectable row at cursor 0

	m, cmd := step(t, m, tea.KeyMsg{Type: tea.KeyEnter})
	if m.state != stateDetail {
		t.Fatalf("enter on a process row should open detail; state = %v", m.state)
	}
	if cmd == nil {
		t.Error("enter should kick off the detail fetch command")
	}
}

func TestUpdateFocusSwitch(t *testing.T) {
	// Tab moves focus from the main list to the side (tree) pane on Processes.
	m, _ := step(t, InitialModel("test"), tea.KeyMsg{Type: tea.KeyTab})
	if m.listFocus != focusSide {
		t.Errorf("tab should move focus to the side pane, got %v", m.listFocus)
	}
}

func TestUpdateListNavMovesCursor(t *testing.T) {
	m := InitialModel("test")
	m.processes = []model.Process{{PID: 1}, {PID: 2}, {PID: 3}}
	m.filterProcesses()

	m, _ = step(t, m, tea.KeyMsg{Type: tea.KeyDown})
	if got := m.table.Cursor(); got != 1 {
		t.Errorf("down should move the cursor to 1, got %d", got)
	}
}

func TestUpdateTogglesShowAllPorts(t *testing.T) {
	m := InitialModel("test")
	m.activeTab = tabPorts
	if m.showAllPorts {
		t.Fatal("precondition: showAllPorts should start false")
	}
	m, _ = step(t, m, keyRunes("a"))
	if !m.showAllPorts {
		t.Error("'a' on the Ports tab should toggle showAllPorts on")
	}
}

func TestUpdateSlashFocusesFilter(t *testing.T) {
	m, cmd := step(t, InitialModel("test"), keyRunes("/"))
	if !m.input.Focused() {
		t.Error("'/' should focus the process filter input")
	}
	if cmd == nil {
		t.Error("'/' should return the cursor-blink command")
	}

	// Typing into the focused filter narrows the list.
	m.processes = []model.Process{{PID: 1, Command: "nginx"}, {PID: 2, Command: "redis"}}
	m, _ = step(t, m, keyRunes("n"))
	if m.input.Value() != "n" {
		t.Fatalf("filter value = %q, want \"n\"", m.input.Value())
	}
	if len(m.filtered) != 1 || m.filtered[0].Command != "nginx" {
		t.Errorf("typing 'n' should narrow to [nginx], got %v", m.filtered)
	}
}

func TestHandleSortKey(t *testing.T) {
	t.Run("processes new column defaults to desc, re-press toggles", func(t *testing.T) {
		m := InitialModel("test") // starts sorted by mem
		m2, _, handled := m.handleSortKey(keyRunes("p"))
		if !handled || m2.sortCol != "pid" || !m2.sortDesc {
			t.Fatalf("'p' => handled=%v col=%q desc=%v, want true/pid/true", handled, m2.sortCol, m2.sortDesc)
		}
		m3, _, _ := m2.handleSortKey(keyRunes("p"))
		if m3.sortDesc {
			t.Errorf("re-pressing 'p' should flip to ascending")
		}
	})

	t.Run("ports column defaults to asc", func(t *testing.T) {
		m := InitialModel("test")
		m.activeTab = tabPorts
		m2, _, handled := m.handleSortKey(keyRunes("s"))
		if !handled || m2.sortPortCol != "state" || m2.sortPortDesc {
			t.Errorf("'s' on Ports => handled=%v col=%q desc=%v, want true/state/false", handled, m2.sortPortCol, m2.sortPortDesc)
		}
	})

	t.Run("containers and locks map keys to columns", func(t *testing.T) {
		mc := InitialModel("test")
		mc.activeTab = tabContainers
		if m2, _, h := mc.handleSortKey(keyRunes("r")); !h || m2.sortContainerCol != "runtime" {
			t.Errorf("'r' on Containers => handled=%v col=%q, want true/runtime", h, m2.sortContainerCol)
		}

		ml := InitialModel("test")
		ml.activeTab = tabLocks
		if m2, _, h := ml.handleSortKey(keyRunes("f")); !h || m2.sortLockCol != "path" {
			t.Errorf("'f' on Locks => handled=%v col=%q, want true/path", h, m2.sortLockCol)
		}
	})

	t.Run("irrelevant key is not handled", func(t *testing.T) {
		m := InitialModel("test")
		if _, _, handled := m.handleSortKey(keyRunes("z")); handled {
			t.Error("'z' is not a sort key and must report not-handled")
		}
	})
}

func TestUpdateResizeSetsDimsAndCmdColumn(t *testing.T) {
	// A wide window leaves room for the Command column.
	m, _ := step(t, InitialModel("test"), tea.WindowSizeMsg{Width: 200, Height: 50})
	if m.width != 200 || m.height != 50 {
		t.Errorf("dims = %dx%d, want 200x50", m.width, m.height)
	}
	if !m.showCmdCol {
		t.Error("a 200-wide window should show the Command column")
	}

	// A narrow window hides it.
	m, _ = step(t, InitialModel("test"), tea.WindowSizeMsg{Width: 40, Height: 20})
	if m.showCmdCol {
		t.Error("a 40-wide window should hide the Command column")
	}
}

func TestUpdateProcessListMessage(t *testing.T) {
	m := InitialModel("test")
	m, cmd := step(t, m, []model.Process{{PID: 10, Command: "a"}, {PID: 20, Command: "b"}})
	if len(m.processes) != 2 || len(m.filtered) != 2 {
		t.Fatalf("process list message should populate processes/filtered, got %d/%d", len(m.processes), len(m.filtered))
	}
	if cmd == nil {
		t.Error("a non-empty process list should schedule a tree fetch for the selection")
	}
}

func TestUpdateDataListMessages(t *testing.T) {
	m := InitialModel("test")

	m, _ = step(t, m, []model.OpenPort{{Port: 80, Protocol: "tcp", State: "LISTEN"}})
	if len(m.ports) != 1 {
		t.Errorf("port list message: ports = %d, want 1", len(m.ports))
	}

	m, _ = step(t, m, []*model.ContainerMatch{{Name: "web"}})
	if len(m.containers) != 1 {
		t.Errorf("container list message: containers = %d, want 1", len(m.containers))
	}

	m, _ = step(t, m, []*model.LockedFile{{PID: 1, Path: "/a"}})
	if len(m.locks) != 1 {
		t.Errorf("lock list message: locks = %d, want 1", len(m.locks))
	}
}

func TestUpdateResultMessageSetsDetail(t *testing.T) {
	m := InitialModel("test")
	res := model.Result{Process: model.Process{PID: 1, Command: "x"}, Ancestry: []model.Process{{PID: 1, Command: "x"}}}
	m, _ = step(t, m, res)
	if m.selectedDetail == nil {
		t.Error("a Result message should populate selectedDetail")
	}
}

func TestUpdateErrorMessageRevertsToList(t *testing.T) {
	m := InitialModel("test")
	m.state = stateDetail
	m, cmd := step(t, m, error(fmt.Errorf("boom")))
	if m.state != stateList {
		t.Errorf("an error should revert to the list view, state = %v", m.state)
	}
	if m.statusMsg == "" {
		t.Error("an error should surface a status message")
	}
	if cmd == nil {
		t.Error("an error should trigger a process refresh")
	}
}

func TestHandleTickRefreshesWhenDue(t *testing.T) {
	m := InitialModel("test")
	old := time.Now().Add(-time.Hour) // long past the cadence -> a tick is due
	m.lastRefresh = old

	nm, cmd := m.handleTick(tickMsg(time.Now()))
	m = nm.(MainModel)
	if !m.lastRefresh.After(old) {
		t.Error("a due tick should advance lastRefresh")
	}
	if cmd == nil {
		t.Error("handleTick must always reschedule the next tick")
	}
}

func TestDetailKeyNavigation(t *testing.T) {
	base := func() MainModel {
		m := InitialModel("test")
		m.state = stateDetail
		m.selectedDetail = &model.Result{Process: model.Process{PID: 1}}
		return m
	}

	t.Run("esc returns to list", func(t *testing.T) {
		m, cmd := step(t, base(), tea.KeyMsg{Type: tea.KeyEsc})
		if m.state != stateList || m.selectedDetail != nil {
			t.Errorf("esc should return to list and clear detail; state=%v detail=%v", m.state, m.selectedDetail)
		}
		if cmd == nil {
			t.Error("leaving detail should refresh the list")
		}
	})

	t.Run("tab toggles detail/env focus", func(t *testing.T) {
		m := base()
		if m.detailFocus != focusDetail {
			t.Fatalf("precondition: detailFocus = %v, want focusDetail", m.detailFocus)
		}
		m, _ = step(t, m, tea.KeyMsg{Type: tea.KeyTab})
		if m.detailFocus != focusEnv {
			t.Errorf("tab should move focus to the env pane, got %v", m.detailFocus)
		}
	})
}
