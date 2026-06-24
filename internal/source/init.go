package source

import (
	"path/filepath"
	"strconv"
	"strings"

	"github.com/pranshuparmar/witr/pkg/model"
)

// detectInit checks if the process is a direct descendant of the system's root
// process: PID 1 (init/systemd/OpenRC/... on Unix) or PID 4 "System" (the kernel
// on Windows). It acts as a catch-all so kernel/system processes resolve to an
// init source rather than appearing unsupervised.
func detectInit(ancestry []model.Process) *model.Source {
	if len(ancestry) == 0 {
		return nil
	}

	root := ancestry[0]
	isWindowsKernel := root.PID == 4 && strings.EqualFold(root.Command, "System")
	if root.PID != 1 && !isWindowsKernel {
		return nil
	}

	// Check if there's any shell in the chain between root and target.
	// If there is a shell, it's likely a manual command run by a user or script, not a pure service.

	hasShell := false
	for i := 1; i < len(ancestry)-1; i++ {
		name := strings.ToLower(filepath.Base(ancestry[i].Command))
		if isShell(name) {
			hasShell = true
			break
		}
	}

	if !hasShell {
		// Use the actual PID 1 command name (e.g., "openrc-init", "runit-init",
		// "init", "systemd") instead of always reporting "init".
		initName := root.Command
		if initName == "" {
			initName = "init"
		}
		src := &model.Source{
			Type: model.SourceInit,
			Name: initName,
			Details: map[string]string{
				"pid":  strconv.Itoa(root.PID),
				"comm": root.Command,
			},
		}
		if isWindowsKernel {
			src.Description = "Windows kernel (System process)"
		}
		return src
	}

	return nil
}
