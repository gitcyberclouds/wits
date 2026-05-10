//go:build windows

package proc

import (
	"context"
	"errors"
	"os/exec"
	"sync/atomic"
	"time"
)

// powerShellTimeout caps every PowerShell/CIM/WMI invocation. WMI providers
// can stall indefinitely (corrupt repository, EDR injection, slow LSASS) and
// without a deadline witr would hang forever — see issue #192.
//
// Five seconds is long enough for a healthy CIM call (typically <1s) and
// short enough that a stuck call doesn't visibly freeze the UI.
const powerShellTimeout = 5 * time.Second

// wmiTimedOut is set whenever a PowerShell/CIM call we made hit its deadline.
// The renderer reads this at the end of a run and prints a single advisory
// line so the user knows the report may be missing optional sections like
// service detection or extended memory stats.
var wmiTimedOut atomic.Bool

// runPowerShell executes the given PowerShell script with a bounded deadline
// and returns its stdout. On timeout it sets the wmiTimedOut flag so the
// renderer can surface a hint, and returns a context.DeadlineExceeded error.
func runPowerShell(script string) ([]byte, error) {
	ctx, cancel := context.WithTimeout(context.Background(), powerShellTimeout)
	defer cancel()

	out, err := exec.CommandContext(ctx, "powershell", "-NoProfile", "-NonInteractive", script).Output()
	if errors.Is(ctx.Err(), context.DeadlineExceeded) {
		wmiTimedOut.Store(true)
		return nil, ctx.Err()
	}
	return out, err
}

// WMITimedOut reports whether any PowerShell/CIM call hit its deadline during
// this run. The renderer uses this to add a single advisory line at the end
// of the report.
func WMITimedOut() bool {
	return wmiTimedOut.Load()
}
