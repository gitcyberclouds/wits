package tui

import "github.com/charmbracelet/lipgloss"

// Theme palette.
//
// Foreground, text, and border colors are AdaptiveColor so the TUI stays
// readable on both light and dark terminals: lipgloss picks Light or Dark at
// render time from the detected background. The Dark values match the original
// scheme, so dark terminals look unchanged; the Light values are darker
// equivalents chosen for contrast on a light background.
//
// Colors that paint their own background (title bar, selected row, tabs) read
// the same on any terminal, so they stay fixed.
var (
	// Accent — table/pane headers, prompts, active borders.
	colorAccent = lipgloss.AdaptiveColor{Light: "#4338ca", Dark: "#5f5fd7"}
	// Inactive borders.
	colorBorderDim = lipgloss.AdaptiveColor{Light: "#c2c2c2", Dark: "#585858"}
	// Inactive panel header text.
	colorHeaderDim = lipgloss.AdaptiveColor{Light: "#5b616e", Dark: "#bcbcbc"}
	// Secondary / muted text (footer, placeholders, empty states).
	colorMuted = lipgloss.AdaptiveColor{Light: "#6e6e6e", Dark: "#767676"}
	// Error text.
	colorError = lipgloss.AdaptiveColor{Light: "#c0271d", Dark: "#ff5f5f"}
	// Action-menu text.
	colorAmber = lipgloss.AdaptiveColor{Light: "#9a6700", Dark: "#ffdf87"}
	// Confirmation prompt text.
	colorConfirm = lipgloss.AdaptiveColor{Light: "#b45309", Dark: "#ffaf5f"}
	// Ancestry-tree connectors.
	colorTreeConn = lipgloss.AdaptiveColor{Light: "#9333ea", Dark: "#d787ff"}
	// Ancestry-tree target node.
	colorTreeTarget = lipgloss.AdaptiveColor{Light: "#15803d", Dark: "#00d700"}
	// Section labels in the detail / tree panes.
	colorSectionLabel = lipgloss.AdaptiveColor{Light: "#7c3aed", Dark: "#af87ff"}

	// Fixed colors — painted over their own background, so they need no
	// adaptation to the terminal background.
	colorBrandFg   = lipgloss.Color("#FAFAFA")
	colorBrandBg   = lipgloss.Color("#7D56F4")
	colorOnAccent  = lipgloss.Color("#ffffff")
	colorGreenBg   = lipgloss.Color("#22aa22")
	colorIdleTabBg = lipgloss.Color("#767676")
	colorSelectFg  = lipgloss.Color("#ffffaf")
	colorSelectBg  = lipgloss.Color("#5f00d7")
)
