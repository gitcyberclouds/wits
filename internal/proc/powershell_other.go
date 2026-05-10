//go:build !windows

package proc

// WMITimedOut is a no-op on non-Windows platforms; the renderer hides the
// timeout-hint surface entirely when this returns false.
func WMITimedOut() bool { return false }
