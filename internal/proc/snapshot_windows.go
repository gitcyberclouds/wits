//go:build windows

package proc

import (
	"fmt"
	"sync"
	"syscall"
	"time"
	"unsafe"
)

// processSnapshot is the lightweight per-process record returned by ToolHelp32.
type processSnapshot struct {
	PID  int
	PPID int
	Exe  string
}

// Cached enumeration so back-to-back calls (TUI list + ancestry walk + child
// resolution within one render pass) reuse a single kernel snapshot instead of
// spinning up ToolHelp32 repeatedly.
var (
	snapshotCache     []processSnapshot
	snapshotCacheTime time.Time
	snapshotCacheMu   sync.Mutex
	snapshotCacheTTL  = 1 * time.Second
)

// enumerateProcesses returns every running process via the ToolHelp32 API.
// This avoids PowerShell+WMI, which can block indefinitely on machines with
// stalled CIM providers (issue #192).
func enumerateProcesses() ([]processSnapshot, error) {
	snapshotCacheMu.Lock()
	defer snapshotCacheMu.Unlock()

	if snapshotCache != nil && time.Since(snapshotCacheTime) < snapshotCacheTTL {
		return snapshotCache, nil
	}

	snap, _, _ := procCreateToolhelp32Snapshot.Call(uintptr(TH32CS_SNAPPROCESS), 0)
	if syscall.Handle(snap) == syscall.InvalidHandle {
		return nil, fmt.Errorf("CreateToolhelp32Snapshot failed")
	}
	defer syscall.CloseHandle(syscall.Handle(snap))

	var pe32 PROCESSENTRY32
	pe32.Size = uint32(unsafe.Sizeof(pe32))

	ret, _, _ := procProcess32First.Call(snap, uintptr(unsafe.Pointer(&pe32)))
	if ret == 0 {
		return nil, fmt.Errorf("Process32First failed")
	}

	var out []processSnapshot
	for {
		out = append(out, processSnapshot{
			PID:  int(pe32.ProcessID),
			PPID: int(pe32.ParentProcessID),
			Exe:  syscall.UTF16ToString(pe32.ExeFile[:]),
		})
		ret, _, _ = procProcess32Next.Call(snap, uintptr(unsafe.Pointer(&pe32)))
		if ret == 0 {
			break
		}
	}

	snapshotCache = out
	snapshotCacheTime = time.Now()
	return out, nil
}
