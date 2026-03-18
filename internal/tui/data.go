package tui

import (
	"cmp"
	"fmt"
	"os"
	"sort"
	"strconv"
	"strings"

	"github.com/charmbracelet/bubbles/table"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/reflow/wrap"
	"github.com/pranshuparmar/witr/internal/output"
	"github.com/pranshuparmar/witr/internal/pipeline"
	"github.com/pranshuparmar/witr/internal/proc"
	"github.com/pranshuparmar/witr/pkg/model"
)

func (m MainModel) refreshProcesses() tea.Cmd {
	return func() tea.Msg {
		procs, err := proc.ListProcesses()
		if err != nil {
			return err
		}

		selfPID := os.Getpid()
		filteredProcs := make([]model.Process, 0, len(procs))
		for _, p := range procs {
			if p.PID == selfPID {
				continue
			}
			if p.PPID == selfPID && (p.Command == "ps" || strings.HasPrefix(p.Command, "ps ")) {
				continue
			}
			filteredProcs = append(filteredProcs, p)
		}
		return filteredProcs
	}
}

func (m MainModel) refreshPorts() tea.Cmd {
	return func() tea.Msg {
		ports, err := proc.ListOpenPorts()
		if err != nil {
			return nil
		}
		return ports
	}
}

func (m MainModel) fetchTree(p model.Process) tea.Cmd {
	return func() tea.Msg {
		res, err := pipeline.AnalyzePID(pipeline.AnalyzeConfig{
			PID:     p.PID,
			Verbose: false,
			Tree:    true,
		})
		if err != nil {
			return treeMsg(model.Result{
				Process: p,
			})
		}
		return treeMsg(res)
	}
}

func (m MainModel) fetchProcessDetail(pid int) tea.Cmd {
	return func() tea.Msg {
		res, err := pipeline.AnalyzePID(pipeline.AnalyzeConfig{
			PID:     pid,
			Verbose: true,
			Tree:    true,
		})
		if err != nil {
			return err
		}
		return res
	}
}

type processSorter struct {
	procs []model.Process
	keys  []string // pre-computed lowercase keys (nil for non-string sorts)
	col   string
	desc  bool
}

func (s processSorter) Len() int { return len(s.procs) }

func (s processSorter) Swap(i, j int) {
	s.procs[i], s.procs[j] = s.procs[j], s.procs[i]
	if s.keys != nil {
		s.keys[i], s.keys[j] = s.keys[j], s.keys[i]
	}
}

func (s processSorter) Less(i, j int) bool {
	var n int
	switch s.col {
	case "pid":
		n = cmp.Compare(s.procs[i].PID, s.procs[j].PID)
	case "name", "user":
		n = cmp.Compare(s.keys[i], s.keys[j])
	case "cpu":
		n = cmp.Compare(s.procs[i].CPUPercent, s.procs[j].CPUPercent)
	case "time":
		n = cmp.Compare(s.procs[i].StartedAt.UnixNano(), s.procs[j].StartedAt.UnixNano())
	default: // "mem"
		n = cmp.Compare(s.procs[i].MemoryRSS, s.procs[j].MemoryRSS)
	}
	if n == 0 {
		n = cmp.Compare(s.procs[i].PID, s.procs[j].PID)
	}
	if s.desc {
		return n > 0
	}
	return n < 0
}

func (m *MainModel) sortProcesses() {
	s := processSorter{
		procs: m.processes,
		col:   m.sortCol,
		desc:  m.sortDesc,
	}

	if m.sortCol == "name" || m.sortCol == "user" {
		s.keys = make([]string, len(m.processes))
		for i := range m.processes {
			if m.sortCol == "name" {
				s.keys[i] = strings.ToLower(m.processes[i].Command)
			} else {
				s.keys[i] = strings.ToLower(m.processes[i].User)
			}
		}
	}

	sort.Stable(s)
}

func (m *MainModel) sortPorts() {
	sort.Slice(m.ports, func(i, j int) bool {
		var less bool
		switch m.sortPortCol {
		case "port":
			less = m.ports[i].Port < m.ports[j].Port
		case "proto":
			less = strings.ToLower(m.ports[i].Protocol) < strings.ToLower(m.ports[j].Protocol)
		case "addr":
			less = strings.ToLower(m.ports[i].Address) < strings.ToLower(m.ports[j].Address)
		case "state":
			less = strings.ToLower(m.ports[i].State) < strings.ToLower(m.ports[j].State)
		default:
			less = m.ports[i].Port < m.ports[j].Port
		}
		if m.sortPortDesc {
			return !less
		}
		return less
	})
}

func formatBytes(bytes uint64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := uint64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
		if exp >= 5 { //avoid index out of range
			break
		}
	}
	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}

func (m *MainModel) filterProcesses() {
	filter := strings.ToLower(m.input.Value())
	var rows []table.Row

	m.filtered = nil
	for _, p := range m.processes {
		cmd := strings.ToLower(p.Command)

		match := false
		if filter == "" {
			match = true
		} else {
			match = strings.Contains(cmd, filter) ||
				strings.Contains(strconv.Itoa(p.PID), filter) ||
				strings.Contains(strings.ToLower(p.User), filter) ||
				strings.Contains(strings.ToLower(p.Cmdline), filter)
		}

		if match {
			m.filtered = append(m.filtered, p)
			startedStr := p.StartedAt.Format("Jan 02 15:04:05")
			if p.StartedAt.IsZero() {
				startedStr = ""
			}

			row := table.Row{
				fmt.Sprintf("%8d", p.PID),
				p.User,
				p.Command,
				fmt.Sprintf("%6s", fmt.Sprintf("%.1f%%", p.CPUPercent)),
				fmt.Sprintf("%16s", fmt.Sprintf("%s (%.1f%%)", formatBytes(p.MemoryRSS), p.MemoryPercent)),
				startedStr,
			}
			if m.showCmdCol {
				row = append(row, p.Cmdline)
			}
			rows = append(rows, row)
		}
	}
	m.table.SetRows(rows)
}

func centerHeader(title string, width int) string {
	w := lipgloss.Width(title)
	if w >= width {
		return title
	}
	pad := width - w
	left := pad / 2
	return strings.Repeat(" ", left) + title
}

func (m *MainModel) getColumns() []table.Column {
	cols := []table.Column{
		{Title: "PID", Width: 8},
		{Title: "User", Width: 12},
		{Title: "Name", Width: 20},
		{Title: "CPU%", Width: 6},
		{Title: "Mem", Width: 16},
		{Title: "Started", Width: 19},
	}
	if m.showCmdCol {
		cols = append(cols, table.Column{Title: "Command", Width: 50})
	}

	addArrow := func(idx int, key string) {
		if idx < len(cols) && m.sortCol == key {
			if m.sortDesc {
				cols[idx].Title += " ↓"
			} else {
				cols[idx].Title += " ↑"
			}
		}
	}

	addArrow(0, "pid")
	addArrow(1, "user")
	addArrow(2, "name")
	addArrow(3, "cpu")
	addArrow(4, "mem")
	addArrow(5, "time")

	// Center-align numeric column headers
	for _, idx := range []int{0, 3, 4} {
		if idx < len(cols) {
			cols[idx].Title = centerHeader(cols[idx].Title, cols[idx].Width)
		}
	}

	return cols
}

func (m *MainModel) getPortColumns() []table.Column {
	cols := []table.Column{
		{Title: "Port", Width: 6},
		{Title: "Protocol", Width: 10},
		{Title: "Address", Width: 30},
		{Title: "State", Width: 20},
	}

	addArrow := func(idx int, key string) {
		if m.sortPortCol == key {
			if m.sortPortDesc {
				cols[idx].Title += " ↓"
			} else {
				cols[idx].Title += " ↑"
			}
		}
	}

	addArrow(0, "port")
	addArrow(1, "proto")
	addArrow(2, "addr")
	addArrow(3, "state")

	// Center-align the Port column header
	cols[0].Title = centerHeader(cols[0].Title, cols[0].Width)

	return cols
}

func (m *MainModel) updatePortTable() {
	m.sortPorts()

	var rows []table.Row
	filter := strings.ToLower(m.portInput.Value())
	seen := make(map[string]bool)

	existingCols := m.portTable.Columns()
	newCols := m.getPortColumns()
	for i := range existingCols {
		if i < len(newCols) {
			newCols[i].Width = existingCols[i].Width
		}
	}
	m.portTable.SetColumns(newCols)

	procMap := make(map[int]model.Process)
	for _, p := range m.processes {
		procMap[p.PID] = p
	}

	for _, p := range m.ports {
		match := false
		if filter == "" {
			match = true
		} else {
			if strings.Contains(fmt.Sprintf("%d", p.Port), filter) ||
				strings.Contains(strings.ToLower(p.Protocol), filter) ||
				strings.Contains(strings.ToLower(p.Address), filter) ||
				strings.Contains(strings.ToLower(p.State), filter) {
				match = true
			}
		}

		if match {
			if !m.showAllPorts && p.State != "LISTEN" && p.State != "OPEN" {
				continue
			}

			key := fmt.Sprintf("%d|%s|%s|%s", p.Port, p.Protocol, p.Address, p.State)

			if !seen[key] {
				seen[key] = true
				rows = append(rows, table.Row{
					fmt.Sprintf("%6d", p.Port),
					p.Protocol,
					p.Address,
					p.State,
				})
			}
		}
	}

	m.portTable.SetRows(rows)
	m.updatePortDetailsWithMap(procMap)
}

func (m *MainModel) updatePortDetails() {
	procMap := make(map[int]model.Process, len(m.processes))
	for _, p := range m.processes {
		procMap[p.PID] = p
	}
	m.updatePortDetailsWithMap(procMap)
}

func (m *MainModel) updatePortDetailsWithMap(procMap map[int]model.Process) {
	selected := m.portTable.SelectedRow()
	if len(selected) < 4 {
		m.portDetailTable.SetRows(nil)
		return
	}

	portStr := strings.TrimSpace(selected[0])
	protocol := selected[1]
	address := selected[2]
	state := selected[3]

	port, _ := strconv.Atoi(portStr)

	var rows []table.Row
	seen := make(map[int]bool)

	for _, p := range m.ports {
		if p.Port == port && p.Protocol == protocol && p.Address == address && p.State == state {
			if !seen[p.PID] {
				seen[p.PID] = true
				if proc, ok := procMap[p.PID]; ok {
					cmd := proc.Cmdline
					cols := m.portDetailTable.Columns()
					if len(cols) > 3 {
						width := cols[3].Width
						if width > 3 && len(cmd) > width {
							cmd = cmd[:width-3] + "..."
						}
					}
					rows = append(rows, table.Row{
						fmt.Sprintf("%8d", proc.PID),
						proc.User,
						proc.Command,
						cmd,
					})
				} else {
					rows = append(rows, table.Row{
						fmt.Sprintf("%8d", p.PID),
						"???",
						"???",
						"???",
					})
				}
			}
		}
	}
	m.portDetailTable.SetRows(rows)
	if len(rows) > 0 {
		m.portDetailTable.SetCursor(0)
	}
}

func (m *MainModel) updateDetailViewport() {
	if m.selectedDetail == nil {
		return
	}
	res := *m.selectedDetail
	var b strings.Builder

	output.RenderStandard(&b, res, true, true)

	content := b.String()
	if m.viewport.Width > 0 {
		content = wrap.String(content, m.viewport.Width)
	}
	m.viewport.SetContent(content)
}

func (m *MainModel) updateEnvViewport() {
	if m.selectedDetail == nil {
		return
	}
	res := *m.selectedDetail
	var b strings.Builder

	if len(res.Process.Env) > 0 {
		for _, env := range res.Process.Env {
			fmt.Fprintf(&b, "%s\n", env)
		}
	} else {
		dimStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#767676"))
		fmt.Fprintf(&b, "%s\n", dimStyle.Render("No environment variables found."))
	}

	content := b.String()
	if m.envViewport.Width > 0 {
		content = wrap.String(content, m.envViewport.Width)
	}
	m.envViewport.SetContent(content)
}

func (m *MainModel) updateTreeViewport(res model.Result) {
	if len(res.Ancestry) == 0 && res.Process.PID == 0 {
		m.treeViewport.SetContent("")
		m.treePIDs = nil
		m.treeCursor = 0
		m.treeResult = nil
		m.treeAncestry = nil
		return
	}

	ancestry := res.Ancestry
	if len(ancestry) == 0 {
		if res.Process.PID > 0 {
			ancestry = []model.Process{res.Process}
		}
	}

	isRefresh := m.treeTargetPID == res.Process.PID

	resCopy := res
	m.treeResult = &resCopy
	m.treeAncestry = ancestry
	m.treeTargetPID = res.Process.PID

	// Build the flat PID list: ancestry then children (max 10)
	var newPIDs []int
	for _, p := range ancestry {
		newPIDs = append(newPIDs, p.PID)
	}
	childLimit := 10
	if len(res.Children) < childLimit {
		childLimit = len(res.Children)
	}
	for i := 0; i < childLimit; i++ {
		newPIDs = append(newPIDs, res.Children[i].PID)
	}

	// Preserve cursor position only on periodic refresh of the same process
	restored := false
	if isRefresh {
		oldCursor := m.treeCursor
		oldPID := 0
		if oldCursor >= 0 && oldCursor < len(m.treePIDs) {
			oldPID = m.treePIDs[oldCursor]
		}
		if oldPID > 0 {
			for i, pid := range newPIDs {
				if pid == oldPID {
					m.treeCursor = i
					restored = true
					break
				}
			}
		}
	}
	m.treePIDs = newPIDs

	if !restored {
		if len(ancestry) > 0 {
			m.treeCursor = len(ancestry) - 1
		} else {
			m.treeCursor = 0
		}
	}

	m.renderTreeContent(res, ancestry)
}

// rerenderTree re-renders the tree viewport with the current cursor position.
func (m *MainModel) rerenderTree() {
	if m.treeResult == nil {
		return
	}
	m.renderTreeContent(*m.treeResult, m.treeAncestry)
}

func (m *MainModel) renderTreeContent(res model.Result, ancestry []model.Process) {
	var b strings.Builder

	magenta := lipgloss.NewStyle().Foreground(lipgloss.Color("#d787ff"))
	green := lipgloss.NewStyle().Foreground(lipgloss.Color("#00d700"))
	highlight := lipgloss.NewStyle().
		Background(lipgloss.Color("#5f00d7")).
		Foreground(lipgloss.Color("#ffffaf"))
	dim := lipgloss.NewStyle().Foreground(lipgloss.Color("#767676"))
	sectionLabel := lipgloss.NewStyle().Foreground(lipgloss.Color("#af87ff")).Bold(true)

	fmt.Fprintf(&b, "%s\n", sectionLabel.Render("Ancestry Tree:"))

	if len(ancestry) == 0 {
		fmt.Fprintf(&b, "  %s\n", dim.Render("No ancestry found"))
	}

	idx := 0
	for i, proc := range ancestry {
		indent := strings.Repeat("  ", i)
		if i > 0 {
			fmt.Fprintf(&b, "%s%s ", indent, magenta.Render("└─"))
		}

		label := fmt.Sprintf("%s (pid %d)", proc.Command, proc.PID)
		if idx == m.treeCursor {
			label = highlight.Render(label)
		} else if i == len(ancestry)-1 {
			label = green.Render(label)
		}
		fmt.Fprintf(&b, "%s\n", label)
		idx++
	}

	children := res.Children
	limit := 10
	count := len(children)
	if count > 0 {
		baseIndent := strings.Repeat("  ", len(ancestry))
		for i, child := range children {
			if i >= limit {
				remaining := count - limit
				fmt.Fprintf(&b, "%s%s ... and %d more\n", baseIndent, magenta.Render("└─"), remaining)
				break
			}
			connector := "├─"
			isLast := (i == count-1) || (i == limit-1 && count <= limit)
			if isLast {
				connector = "└─"
			}

			label := fmt.Sprintf("%s (pid %d)", child.Command, child.PID)
			if idx == m.treeCursor {
				label = highlight.Render(label)
			}
			fmt.Fprintf(&b, "%s%s %s\n", baseIndent, magenta.Render(connector), label)
			idx++
		}
	}

	if res.Process.Cmdline != "" {
		fmt.Fprintf(&b, "\n%s\n%s\n", sectionLabel.Render("Command:"), res.Process.Cmdline)
	}

	content := b.String()
	if m.treeViewport.Width > 0 {
		content = wrap.String(content, m.treeViewport.Width)
	}
	m.treeViewport.SetContent(content)
}
