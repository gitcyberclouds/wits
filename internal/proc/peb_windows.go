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

	// Get PPID and a fallback image name from the snapshot.
	ppid, snapExe, _ := getInfoFromSnapshot(pid)
	info.PPID = ppid

	// Prefer the full image path. Fall back to the snapshot's bare image name
	// for minimal/system processes (System, Memory Compression, vmmemWSL) whose
	// path can't be queried, so the name still resolves instead of being blank.
	exePath := getProcessImageName(handleLimited)
	if exePath == "" {
		exePath = snapExe
	}
	info.Exe = exePath
	if exePath != "" {
		info.CommandLine = filepath.Base(exePath)
	}

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
	info.Env = readEnvironmentBlock(handle, params.Environment)

	return nil
}

func readProcessMemory(handle syscall.Handle, addr uintptr, dest unsafe.Pointer, size uintptr) bool {
	// lpNumberOfBytesRead is a SIZE_T* (pointer-sized: 8 bytes on x64). It MUST
	// be uintptr, not uint32 — a uint32 here lets the kernel write 8 bytes into
	// a 4-byte slot, corrupting adjacent memory and causing nondeterministic
	// crashes far from this call site.
	var read uintptr
	ret, _, _ := procReadProcessMem.Call(
		uintptr(handle),
		addr,
		uintptr(dest),
		size,
		uintptr(unsafe.Pointer(&read)),
	)
	return ret != 0
}

// readEnvironmentBlock reads a process's environment from its PEB. The block is
// a run of "KEY=VALUE\0" entries terminated by an empty entry (a \0\0). It is
// read in chunks until that terminator appears or a read fails, and bounded so a
// corrupt pointer can't drive an unbounded read of remote memory.
func readEnvironmentBlock(handle syscall.Handle, addr uintptr) []string {
	if addr == 0 {
		return nil
	}
	const chunkWords = 2048    // 4 KiB per read
	const maxWords = 64 * 1024 // cap at 128 KiB
	var block []uint16
	for len(block) < maxWords {
		buf := make([]uint16, chunkWords)
		if !readProcessMemory(handle, addr+uintptr(len(block)*2), unsafe.Pointer(&buf[0]), uintptr(len(buf)*2)) {
			break
		}
		block = append(block, buf...)
		if envBlockEnd(block) >= 0 {
			break
		}
	}
	return parseEnvBlock(block)
}

// envBlockEnd returns the index of the \0\0 terminator, or -1 if not yet read.
func envBlockEnd(block []uint16) int {
	for i := 0; i+1 < len(block); i++ {
		if block[i] == 0 && block[i+1] == 0 {
			return i
		}
	}
	return -1
}

// parseEnvBlock splits the environment block into "KEY=VALUE" entries, stopping
// at the empty entry that terminates it.
func parseEnvBlock(block []uint16) []string {
	var env []string
	start := 0
	for i := 0; i < len(block); i++ {
		if block[i] != 0 {
			continue
		}
		if i == start { // empty entry: end of block
			break
		}
		env = append(env, syscall.UTF16ToString(block[start:i]))
		start = i + 1
	}
	return env
}

func readUnicodeString(handle syscall.Handle, us unicodeString) string {
	// us.Length is a byte count. Read only whole uint16 code units, and never
	// more bytes than the destination buffer holds: a malformed (odd or partial)
	// Length from an incomplete PEB read must not overrun buf.
	n := int(us.Length) / 2
	if n == 0 || us.Buffer == 0 {
		return ""
	}
	buf := make([]uint16, n)
	if !readProcessMemory(handle, us.Buffer, unsafe.Pointer(&buf[0]), uintptr(n*2)) {
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
	procs, err := enumerateProcesses()
	if err != nil {
		return 0, "", err
	}
	for _, p := range procs {
		if p.PID == pid {
			return p.PPID, p.Exe, nil
		}
	}
	return 0, "", fmt.Errorf("process %d not found in snapshot", pid)
}

// processCommandLineInformation is the NtQueryInformationProcess class (60,
// Windows 8.1+) that returns a process's command line.
const processCommandLineInformation = 60

// windowsProcessCmdline returns a process's full command line via
// NtQueryInformationProcess(ProcessCommandLineInformation). The kernel copies
// the command line into our own buffer, so — unlike walking the PEB with
// ReadProcessMemory — there is no remote process-memory access: inaccessible or
// unusual processes return an error, and any anomaly degrades to an empty
// string rather than faulting. Safe to call across the whole process list.
func windowsProcessCmdline(pid int) string {
	handle, err := syscall.OpenProcess(PROCESS_QUERY_LIMITED_INFORMATION, false, uint32(pid))
	if err != nil {
		return ""
	}
	defer syscall.CloseHandle(handle)

	const statusInfoLengthMismatch = 0xC0000004
	bufLen := uint32(4096)
	for attempt := 0; attempt < 2; attempt++ {
		buf := make([]byte, bufLen)
		var retLen uint32
		status, _, _ := procNtQueryInfo.Call(
			uintptr(handle),
			uintptr(processCommandLineInformation),
			uintptr(unsafe.Pointer(&buf[0])),
			uintptr(bufLen),
			uintptr(unsafe.Pointer(&retLen)),
		)
		// Buffer too small: grow to the reported size and retry once.
		if uint32(status) == statusInfoLengthMismatch && retLen > bufLen && retLen <= 1<<20 {
			bufLen = retLen
			continue
		}
		if status != 0 {
			return ""
		}

		// The buffer starts with a UNICODE_STRING whose Buffer points into the
		// same buffer, just past the struct. Validate every offset before use.
		us := (*unicodeString)(unsafe.Pointer(&buf[0]))
		if us.Length == 0 || us.Buffer == 0 {
			return ""
		}
		base := uintptr(unsafe.Pointer(&buf[0]))
		if us.Buffer < base {
			return ""
		}
		offset := us.Buffer - base
		if offset+uintptr(us.Length) > uintptr(len(buf)) {
			return ""
		}
		return syscall.UTF16ToString(unsafe.Slice((*uint16)(unsafe.Pointer(&buf[offset])), int(us.Length)/2))
	}
	return ""
}
