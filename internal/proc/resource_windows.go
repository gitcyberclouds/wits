//go:build windows

package proc

import (
	"sync"
	"syscall"
	"time"
	"unsafe"

	"github.com/pranshuparmar/witr/pkg/model"
)

var procGlobalMemoryStatusEx = modkernel32.NewProc("GlobalMemoryStatusEx")

// memoryStatusEx mirrors MEMORYSTATUSEX. dwLength must be set to sizeof(struct)
// before GlobalMemoryStatusEx is called.
type memoryStatusEx struct {
	Length               uint32
	MemoryLoad           uint32
	TotalPhys            uint64
	AvailPhys            uint64
	TotalPageFile        uint64
	AvailPageFile        uint64
	TotalVirtual         uint64
	AvailVirtual         uint64
	AvailExtendedVirtual uint64
}

var (
	totalPhysOnce sync.Once
	totalPhysVal  uint64
)

// windowsTotalPhysicalMemory returns total physical RAM in bytes. The value is
// fixed for the life of the process, so it's read once and cached.
func windowsTotalPhysicalMemory() uint64 {
	totalPhysOnce.Do(func() {
		var ms memoryStatusEx
		ms.Length = uint32(unsafe.Sizeof(ms))
		if ret, _, _ := procGlobalMemoryStatusEx.Call(uintptr(unsafe.Pointer(&ms))); ret != 0 {
			totalPhysVal = ms.TotalPhys
		}
	})
	return totalPhysVal
}

// windowsMemoryPercent expresses a resident byte count as a percentage of total
// physical RAM.
func windowsMemoryPercent(rss uint64) float64 {
	total := windowsTotalPhysicalMemory()
	if total == 0 {
		return 0
	}
	return float64(rss) / float64(total) * 100.0
}

// windowsProcMetrics opens a process once and returns its resident set size
// (WorkingSetSize, in bytes), lifetime-average CPU%, total CPU time, and start
// time. All are zero values when the handle can't be opened, which is expected
// for protected/system processes without elevation — callers keep the identity
// fields and report zeros rather than failing. It reads only fixed-size kernel
// structures (no remote process-memory access), so it's safe to call across the
// full process list.
func windowsProcMetrics(pid int) (rss uint64, cpu float64, cpuTime time.Duration, started time.Time) {
	handle, err := syscall.OpenProcess(PROCESS_QUERY_LIMITED_INFORMATION, false, uint32(pid))
	if err != nil {
		return 0, 0, 0, time.Time{}
	}
	defer syscall.CloseHandle(handle)

	var pmc processMemoryCountersEx
	pmc.CB = uint32(unsafe.Sizeof(pmc))
	if ret, _, _ := procGetProcessMemoryInfo.Call(
		uintptr(handle),
		uintptr(unsafe.Pointer(&pmc)),
		uintptr(pmc.CB),
	); ret != 0 {
		rss = uint64(pmc.WorkingSetSize)
	}

	var creation, exit, kernel, user syscall.Filetime
	if ret, _, _ := procGetProcessTimes.Call(
		uintptr(handle),
		uintptr(unsafe.Pointer(&creation)),
		uintptr(unsafe.Pointer(&exit)),
		uintptr(unsafe.Pointer(&kernel)),
		uintptr(unsafe.Pointer(&user)),
	); ret != 0 {
		started = time.Unix(0, creation.Nanoseconds())
		cpuTime = filetimeTicksToDuration(kernel) + filetimeTicksToDuration(user)
		wall := time.Since(started)
		if wall > 0 {
			cpu = float64(cpuTime) / float64(wall) * 100.0
		}
	}

	return rss, cpu, cpuTime, started
}

// windowsHealth derives a health status from a process's resident memory and
// total CPU time. Windows has no zombie/stopped equivalent, so it reports the
// resource conditions, matching the Unix >2h CPU and >1GiB RSS thresholds.
func windowsHealth(rss uint64, cpuTime time.Duration) string {
	switch {
	case cpuTime > 2*time.Hour:
		return "high-cpu"
	case rss > 1<<30: // 1 GiB
		return "high-mem"
	default:
		return "healthy"
	}
}

// GetResourceContext returns CPU and memory usage for a process.
//
// CPU usage is the lifetime average — total kernel + user CPU time divided
// by wall-clock time since the process started — not an instantaneous %.
// Memory is the private commit (PrivateUsage).
func GetResourceContext(pid int) *model.ResourceContext {
	handle, err := syscall.OpenProcess(PROCESS_QUERY_LIMITED_INFORMATION, false, uint32(pid))
	if err != nil {
		return nil
	}
	defer syscall.CloseHandle(handle)

	var (
		cpu float64
		mem uint64
	)

	var pmc processMemoryCountersEx
	pmc.CB = uint32(unsafe.Sizeof(pmc))
	if ret, _, _ := procGetProcessMemoryInfo.Call(
		uintptr(handle),
		uintptr(unsafe.Pointer(&pmc)),
		uintptr(pmc.CB),
	); ret != 0 {
		mem = uint64(pmc.PrivateUsage)
	}

	var creation, exit, kernel, user syscall.Filetime
	if ret, _, _ := procGetProcessTimes.Call(
		uintptr(handle),
		uintptr(unsafe.Pointer(&creation)),
		uintptr(unsafe.Pointer(&exit)),
		uintptr(unsafe.Pointer(&kernel)),
		uintptr(unsafe.Pointer(&user)),
	); ret != 0 {
		startTime := time.Unix(0, creation.Nanoseconds())
		wall := time.Since(startTime)
		cpuTime := filetimeTicksToDuration(kernel) + filetimeTicksToDuration(user)
		if wall > 0 {
			cpu = float64(cpuTime) / float64(wall) * 100.0
		}
	}

	return &model.ResourceContext{
		CPUUsage:    cpu,
		MemoryUsage: mem,
	}
}

// filetimeTicksToDuration treats a Filetime as a count of 100-ns ticks (the
// shape kernel/user time take in GetProcessTimes), not as an absolute
// timestamp.
func filetimeTicksToDuration(ft syscall.Filetime) time.Duration {
	ticks := uint64(ft.HighDateTime)<<32 | uint64(ft.LowDateTime)
	return time.Duration(ticks) * 100 * time.Nanosecond
}
