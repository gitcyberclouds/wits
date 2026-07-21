//go:build js

// world_js.go makes the browser a fifth platform for witr. Instead of reading
// /proc, syscalls, or shelling out, the process/port/lock readers return data
// from an in-memory "world" injected by the WASM entry point
// (cmd/witr-wasm). Everything above this layer — ancestry walking, source
// detection, warning generation, and output rendering — is witr's real code,
// running unchanged in the browser.
package proc

import (
	"context"
	"fmt"
	"os/exec"

	"github.com/pranshuparmar/witr/pkg/model"
)

var (
	worldProcs   []model.Process
	worldByPID   = map[int]model.Process{}
	worldLocks   []*model.LockedFile
	worldPorts   []model.OpenPort
	worldSockets = map[int]*model.SocketInfo{}
)

// LoadWorld installs the in-memory machine the readers below serve from.
func LoadWorld(procs []model.Process, locks []*model.LockedFile, ports []model.OpenPort, sockets map[int]*model.SocketInfo) {
	worldProcs = procs
	worldByPID = make(map[int]model.Process, len(procs))
	for _, p := range procs {
		worldByPID[p.PID] = p
	}
	worldLocks = locks
	worldPorts = ports
	if sockets == nil {
		sockets = map[int]*model.SocketInfo{}
	}
	worldSockets = sockets
}

func ReadProcess(pid int) (model.Process, error) {
	if p, ok := worldByPID[pid]; ok {
		return p, nil
	}
	return model.Process{}, fmt.Errorf("process %d not found", pid)
}

func ListProcesses() ([]model.Process, error)       { return worldProcs, nil }
func ListProcessSnapshot() ([]model.Process, error) { return worldProcs, nil }

func GetCmdline(pid int) string {
	if p, ok := worldByPID[pid]; ok {
		return p.Cmdline
	}
	return ""
}

func ReadExtendedInfo(pid int) (model.MemoryInfo, model.IOStats, []string, int, uint64, int, error) {
	p := worldByPID[pid]
	return p.Memory, p.IO, p.FileDescs, p.FDCount, p.FDLimit, p.ThreadCount, nil
}

func GetResourceContext(pid int) *model.ResourceContext { return nil }

func GetFileContext(pid int) *model.FileContext {
	var locked []string
	for _, l := range worldLocks {
		if l != nil && l.PID == pid {
			locked = append(locked, fmt.Sprintf("%s (%s)", l.Path, l.Mode))
		}
	}
	p, ok := worldByPID[pid]
	if !ok {
		if len(locked) == 0 {
			return nil
		}
		return &model.FileContext{LockedFiles: locked}
	}
	if p.FDCount == 0 && len(locked) == 0 {
		return nil
	}
	return &model.FileContext{OpenFiles: p.FDCount, FileLimit: int(p.FDLimit), LockedFiles: locked}
}

func ReadCapabilities(pid int) []string {
	return worldByPID[pid].Capabilities
}

func ResolveChildren(pid int) ([]model.Process, error) {
	var kids []model.Process
	for _, p := range worldProcs {
		if p.PPID == pid && p.PID != pid {
			kids = append(kids, p)
		}
	}
	return kids, nil
}

func ListOpenPorts() ([]model.OpenPort, error) { return worldPorts, nil }

func GetSocketStateForPort(port int) *model.SocketInfo { return worldSockets[port] }

func ListLockedFiles() []*model.LockedFile  { return worldLocks }
func ListAllOpenFiles() []*model.LockedFile { return worldLocks }

// commandAsOriginalUser is only used to drop privileges before shelling out to
// docker/systemctl on real systems — never reached under WASM, but the symbol
// must exist for the package to compile.
func commandAsOriginalUser(ctx context.Context, bin string, args ...string) *exec.Cmd {
	return exec.CommandContext(ctx, bin, args...)
}
