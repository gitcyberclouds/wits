//go:build windows

package target

import (
	"fmt"
	"path/filepath"
	"runtime"
	"syscall"
	"unsafe"
)

// Restart Manager (rstrtmgr.dll) enumerates the processes holding a file open.
// The kernel copies results into our buffer, so there is no remote
// process-memory access.
var (
	modRstrtmgr             = syscall.NewLazyDLL("rstrtmgr.dll")
	procRmStartSession      = modRstrtmgr.NewProc("RmStartSession")
	procRmRegisterResources = modRstrtmgr.NewProc("RmRegisterResources")
	procRmGetList           = modRstrtmgr.NewProc("RmGetList")
	procRmEndSession        = modRstrtmgr.NewProc("RmEndSession")
)

const (
	cchRmSessionKey = 32  // CCH_RM_SESSION_KEY
	errorMoreData   = 234 // ERROR_MORE_DATA
)

type rmUniqueProcess struct {
	ProcessID        uint32
	ProcessStartTime syscall.Filetime
}

// rmProcessInfo mirrors RM_PROCESS_INFO exactly. The array sizes are
// CCH_RM_MAX_APP_NAME+1 (256) and CCH_RM_MAX_SVC_NAME+1 (64); a wrong size would
// misread the kernel-filled buffer (see TestRmProcessInfoSize).
type rmProcessInfo struct {
	Process          rmUniqueProcess
	AppName          [256]uint16
	ServiceShortName [64]uint16
	ApplicationType  uint32
	AppStatus        uint32
	TSSessionID      uint32
	Restartable      int32
}

func ResolveFile(path string) ([]int, error) {
	abs, err := filepath.Abs(path)
	if err != nil {
		abs = path
	}
	pathPtr, err := syscall.UTF16PtrFromString(abs)
	if err != nil {
		return nil, fmt.Errorf("invalid path %q: %w", path, err)
	}

	var session uint32
	sessionKey := make([]uint16, cchRmSessionKey+1)
	if ret, _, _ := procRmStartSession.Call(
		uintptr(unsafe.Pointer(&session)),
		0,
		uintptr(unsafe.Pointer(&sessionKey[0])),
	); ret != 0 {
		return nil, fmt.Errorf("RmStartSession failed (%d)", ret)
	}
	defer procRmEndSession.Call(uintptr(session))

	// rgsFileNames is an array of WCHAR* — &pathPtr is a one-element array.
	if ret, _, _ := procRmRegisterResources.Call(
		uintptr(session),
		1, // nFiles
		uintptr(unsafe.Pointer(&pathPtr)),
		0, 0, // nApplications, rgApplications
		0, 0, // nServices, rgsServiceNames
	); ret != 0 {
		return nil, fmt.Errorf("RmRegisterResources failed (%d)", ret)
	}
	runtime.KeepAlive(pathPtr)

	// First RmGetList sizes the buffer (NULL list).
	var needed, count, rebootReasons uint32
	ret, _, _ := procRmGetList.Call(
		uintptr(session),
		uintptr(unsafe.Pointer(&needed)),
		uintptr(unsafe.Pointer(&count)),
		0,
		uintptr(unsafe.Pointer(&rebootReasons)),
	)
	if ret != 0 && ret != errorMoreData {
		return nil, fmt.Errorf("RmGetList failed (%d)", ret)
	}
	if needed == 0 {
		return nil, fmt.Errorf("no process is holding %s", path)
	}

	infos := make([]rmProcessInfo, needed)
	count = needed
	if ret, _, _ := procRmGetList.Call(
		uintptr(session),
		uintptr(unsafe.Pointer(&needed)),
		uintptr(unsafe.Pointer(&count)),
		uintptr(unsafe.Pointer(&infos[0])),
		uintptr(unsafe.Pointer(&rebootReasons)),
	); ret != 0 {
		return nil, fmt.Errorf("RmGetList failed (%d)", ret)
	}

	var pids []int
	seen := make(map[int]bool)
	for i := 0; i < int(count) && i < len(infos); i++ {
		pid := int(infos[i].Process.ProcessID)
		if pid > 0 && !seen[pid] {
			pids = append(pids, pid)
			seen[pid] = true
		}
	}
	if len(pids) == 0 {
		return nil, fmt.Errorf("no process is holding %s", path)
	}
	return pids, nil
}
