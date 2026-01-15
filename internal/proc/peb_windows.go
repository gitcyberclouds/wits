//go:build windows

package proc

import (
	"fmt"
	"path/filepath"
	"syscall"
	"time"
	"unsafe"
)

// Win32 API constants and structures
const (
	PROCESS_QUERY_INFORMATION         = 0x0400
	PROCESS_VM_READ                   = 0x0010
	PROCESS_QUERY_LIMITED_INFORMATION = 0x1000

	TH32CS_SNAPPROCESS = 0x00000002
)

var (
	modntdll                      = syscall.NewLazyDLL("ntdll.dll")
	procNtQueryInfo               = modntdll.NewProc("NtQueryInformationProcess")
	modkernel32                   = syscall.NewLazyDLL("kernel32.dll")
	procReadProcessMem            = modkernel32.NewProc("ReadProcessMemory")
	procGetProcessTimes           = modkernel32.NewProc("GetProcessTimes")
	procQueryFullProcessImageName = modkernel32.NewProc("QueryFullProcessImageNameW")
	procCreateToolhelp32Snapshot  = modkernel32.NewProc("CreateToolhelp32Snapshot")
	procProcess32First            = modkernel32.NewProc("Process32FirstW")
	procProcess32Next             = modkernel32.NewProc("Process32NextW")
)

type processBasicInformation struct {
	ExitStatus                   uintptr
	PebBaseAddress               uintptr
	AffinityMask                 uintptr
	BasePriority                 uintptr
	UniqueProcessId              uintptr
	InheritedFromUniqueProcessId uintptr
}

type unicodeString struct {
	Length        uint16
	MaximumLength uint16
	Buffer        uintptr
}

// Partial RTL_USER_PROCESS_PARAMETERS
type rtlUserProcessParameters struct {
	Reserved1              [16]byte
	Reserved2              [5]uintptr
	CurrentDirectoryPath   unicodeString
	CurrentDirectoryHandle uintptr
	DllPath                unicodeString
	ImagePathName          unicodeString
	CommandLine            unicodeString
	Environment            uintptr
}

type PROCESSENTRY32 struct {
	Size            uint32
	CntUsage        uint32
	ProcessID       uint32
	DefaultHeapID   uintptr
	ModuleID        uint32
	CntThreads      uint32
	ParentProcessID uint32
	PriClassBase    int32
	Flags           uint32
	ExeFile         [260]uint16
}

type Win32ProcessInfo struct {
	PPID        int
	CommandLine string
	Exe         string
	Cwd         string
	Env         []string
	StartedAt   time.Time
}

func GetProcessDetailedInfo(pid int) (Win32ProcessInfo, error) {
	var info Win32ProcessInfo

	// 1. Try Full Access (Query Info + VM Read)
	handle, err := syscall.OpenProcess(PROCESS_QUERY_INFORMATION|PROCESS_VM_READ, false, uint32(pid))
	if err == nil {
		defer syscall.CloseHandle(handle)
		err := getFullProcessInfo(handle, pid, &info)
		if err == nil {
			return info, nil
		}
		// If getFullProcessInfo fails (e.g. PEB read error), fall through to limited
	}

	// 2. Fallback: Try Limited Access (Query Limited Info)
	// This allows getting Exe Path and Start Time for elevated processes from standard user.
	handleLimited, err := syscall.OpenProcess(PROCESS_QUERY_LIMITED_INFORMATION, false, uint32(pid))
	if err != nil {
		// Fallback: If we can't open the process (Access Denied), try getting basic info from the snapshot.
		ppid, exe, snapErr := getInfoFromSnapshot(pid)
		if snapErr == nil {
			info.PPID = ppid
			info.Exe = exe
			info.CommandLine = exe
			return info, nil
		}
		return info, err
	}
	defer syscall.CloseHandle(handleLimited)

	// Get Start Time
	info.StartedAt = getProcessStartTime(handleLimited)

	// Get Exe Path via QueryFullProcessImageName (Kernel32)
	exePath := getProcessImageName(handleLimited)
	info.Exe = exePath
	// Default CommandLine to Exe name if we can't read memory
	if exePath != "" {
		info.CommandLine = filepath.Base(exePath)
	}

	// Get PPID via Snapshot (since we can't query it from process handle easily without full rights/classes)
	ppid, _, _ := getInfoFromSnapshot(pid)
	info.PPID = ppid

	// Cwd and Env are unavailable without VM_READ
	info.Cwd = ""
	info.Env = []string{}

	return info, nil
}

func getFullProcessInfo(handle syscall.Handle, pid int, info *Win32ProcessInfo) error {
	info.StartedAt = getProcessStartTime(handle)

	var pbi processBasicInformation
	var returnLength uint32
	status, _, _ := procNtQueryInfo.Call(
		uintptr(handle),
		0, // ProcessBasicInformation
		uintptr(unsafe.Pointer(&pbi)),
		uintptr(unsafe.Sizeof(pbi)),
		uintptr(unsafe.Pointer(&returnLength)),
	)

	if status != 0 {
		return fmt.Errorf("NtQueryInformationProcess failed with status %x", status)
	}

	info.PPID = int(pbi.InheritedFromUniqueProcessId)

	if pbi.PebBaseAddress == 0 {
		return fmt.Errorf("PEB Base Address is 0")
	}

	// Read PEB
	var pebPtr uintptr
	paramsOffset := uintptr(0x20)
	if unsafe.Sizeof(uintptr(0)) == 4 {
		paramsOffset = 0x10
	}

	if !readProcessMemory(handle, pbi.PebBaseAddress+paramsOffset, unsafe.Pointer(&pebPtr), unsafe.Sizeof(pebPtr)) {
		return fmt.Errorf("failed to read PEB ProcessParameters address")
	}

	var params rtlUserProcessParameters
	if !readProcessMemory(handle, pebPtr, unsafe.Pointer(&params), unsafe.Sizeof(params)) {
		return fmt.Errorf("failed to read ProcessParameters struct")
	}

	info.Cwd = readUnicodeString(handle, params.CurrentDirectoryPath)
	info.CommandLine = readUnicodeString(handle, params.CommandLine)
	info.Exe = readUnicodeString(handle, params.ImagePathName)
	info.Env = []string{}

	return nil
}

func readProcessMemory(handle syscall.Handle, addr uintptr, dest unsafe.Pointer, size uintptr) bool {
	var read uint32
	ret, _, _ := procReadProcessMem.Call(
		uintptr(handle),
		addr,
		uintptr(dest),
		size,
		uintptr(unsafe.Pointer(&read)),
	)
	return ret != 0
}

func readUnicodeString(handle syscall.Handle, us unicodeString) string {
	if us.Length == 0 {
		return ""
	}
	buf := make([]uint16, us.Length/2)
	if !readProcessMemory(handle, us.Buffer, unsafe.Pointer(&buf[0]), uintptr(us.Length)) {
		return ""
	}
	return syscall.UTF16ToString(buf)
}

func getProcessStartTime(handle syscall.Handle) time.Time {
	var creation, exit, kernel, user syscall.Filetime
	ret, _, _ := procGetProcessTimes.Call(
		uintptr(handle),
		uintptr(unsafe.Pointer(&creation)),
		uintptr(unsafe.Pointer(&exit)),
		uintptr(unsafe.Pointer(&kernel)),
		uintptr(unsafe.Pointer(&user)),
	)
	if ret == 0 {
		return time.Time{}
	}
	return time.Unix(0, creation.Nanoseconds())
}

func getProcessImageName(handle syscall.Handle) string {
	buf := make([]uint16, 1024)
	size := uint32(len(buf))
	// QueryFullProcessImageNameW(hProcess, 0, lpExeName, lpdwSize)
	ret, _, _ := procQueryFullProcessImageName.Call(
		uintptr(handle),
		0,
		uintptr(unsafe.Pointer(&buf[0])),
		uintptr(unsafe.Pointer(&size)),
	)
	if ret == 0 {
		return ""
	}
	return syscall.UTF16ToString(buf[:size])
}

func getInfoFromSnapshot(pid int) (int, string, error) {
	// CreateToolhelp32Snapshot
	snap, _, _ := procCreateToolhelp32Snapshot.Call(uintptr(TH32CS_SNAPPROCESS), 0)
	if syscall.Handle(snap) == syscall.InvalidHandle {
		return 0, "", fmt.Errorf("failed to create snapshot")
	}
	defer syscall.CloseHandle(syscall.Handle(snap))

	var pe32 PROCESSENTRY32
	pe32.Size = uint32(unsafe.Sizeof(pe32))

	// Process32First
	ret, _, _ := procProcess32First.Call(snap, uintptr(unsafe.Pointer(&pe32)))
	if ret == 0 {
		return 0, "", fmt.Errorf("failed to get first process")
	}

	for {
		if int(pe32.ProcessID) == pid {
			exe := syscall.UTF16ToString(pe32.ExeFile[:])
			return int(pe32.ParentProcessID), exe, nil
		}
		// Process32Next
		ret, _, _ = procProcess32Next.Call(snap, uintptr(unsafe.Pointer(&pe32)))
		if ret == 0 {
			break
		}
	}
	return 0, "", fmt.Errorf("process %d not found in snapshot", pid)
}
