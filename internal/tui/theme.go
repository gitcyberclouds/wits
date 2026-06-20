package tui

import "github.com/charmbracelet/lipgloss"

// Theme palette.
//
// Centralizes the TUI's colors in one place instead of scattering hardcoded
// hex literals across the table, tree, detail, and view code. Values match the
// original scheme, so the rendered output is unchanged.
var (
	// Accent — table/pane headers, prompts, active borders.
	colorAccent = lipgloss.Color("#5f5fd7")
	// Inactive borders.
	colorBorderDim = lipgloss.Color("#585858")
	// Inactive panel header text.
	colorHeaderDim = lipgloss.Color("#bcbcbc")
	// Secondary / muted text (footer, placeholders, empty states).
	colorMuted = lipgloss.Color("#767676")
	// Error text.
	colorError = lipgloss.Color("#ff5f5f")
	// Action-menu text.
	colorAmber = lipgloss.Color("#ffdf87")
	// Confirmation prompt text.
	colorConfirm = lipgloss.Color("#ffaf5f")
	// Ancestry-tree connectors.
	colorTreeConn = lipgloss.Color("#d787ff")
	// Ancestry-tree target node.
	colorTreeTarget = lipgloss.Color("#00d700")
	// Section labels in the detail / tree panes.
	colorSectionLabel = lipgloss.Color("#af87ff")

	// Colors painted over their own background (title bar, selected row, tabs).
	colorBrandFg   = lipgloss.Color("#FAFAFA")
	colorBrandBg   = lipgloss.Color("#7D56F4")
	colorOnAccent  = lipgloss.Color("#ffffff")
	colorGreenBg   = lipgloss.Color("#22aa22")
	colorIdleTabBg = lipgloss.Color("#767676")
	colorSelectFg  = lipgloss.Color("#ffffaf")
	colorSelectBg  = lipgloss.Color("#5f00d7")
)
