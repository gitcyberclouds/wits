//go:build windows

package target

import (
	"testing"
	"unsafe"
)

// rmUniqueProcess / rmProcessInfo must match the Win32 structs byte-for-byte or
// RmGetList misreads the kernel-filled buffer. RM_PROCESS_INFO = 12 (unique
// process) + 512 (app name) + 128 (service name) + 16 (four 32-bit fields) = 668.
func TestRmProcessInfoSize(t *testing.T) {
	if got := unsafe.Sizeof(rmUniqueProcess{}); got != 12 {
		t.Errorf("sizeof(rmUniqueProcess) = %d, want 12", got)
	}
	if got := unsafe.Sizeof(rmProcessInfo{}); got != 668 {
		t.Errorf("sizeof(rmProcessInfo) = %d, want 668", got)
	}
}
