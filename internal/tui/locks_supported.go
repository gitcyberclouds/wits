//go:build !windows

package tui

// locksTabEnabled controls whether the Locks tab is shown in the TUI.
// Hidden on Windows where there's no system-wide lock table.
const locksTabEnabled = true
