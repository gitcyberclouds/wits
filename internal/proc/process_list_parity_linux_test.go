//go:build linux

package proc

import (
	"os"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
	"testing"

	"github.com/pranshuparmar/witr/pkg/model"
)

func findListedPID(procs []model.Process, pid int) *model.Process {
	for i := range procs {
		if procs[i].PID == pid {
			return &procs[i]
		}
	}
	return nil
}

// TestListProcessesSelf checks the /proc-based ListProcesses (which replaced the
// `ps -axo` fork) produces self with plausible columns, and that the values it
// computes from /proc track what ps reports.
func TestListProcessesSelf(t *testing.T) {
	procs, err := ListProcesses()
	if err != nil {
		t.Fatalf("ListProcesses: %v", err)
	}

	me := findListedPID(procs, os.Getpid())
	if me == nil {
		t.Fatalf("ListProcesses did not include self (pid %d)", os.Getpid())
	}
	if me.MemoryRSS == 0 {
		t.Error("self RSS should be > 0")
	}
	if me.Command == "" {
		t.Error("self Command should be non-empty")
	}
	if me.User == "" {
		t.Error("self User should be non-empty")
	}
	if me.PPID == 0 {
		t.Error("self PPID should be non-zero")
	}
	if me.CPUPercent < 0 || me.CPUPercent > float64(runtime.NumCPU())*100+1 {
		t.Errorf("self CPU%% = %v is implausible", me.CPUPercent)
	}

	// RSS parity with ps (50% tolerance — resident memory drifts between the two reads).
	if out, err := exec.Command("ps", "-p", strconv.Itoa(os.Getpid()), "-o", "rss=").Output(); err == nil {
		if psKB, perr := strconv.ParseFloat(strings.TrimSpace(string(out)), 64); perr == nil && psKB > 0 {
			psRSS := psKB * 1024
			if ratio := float64(me.MemoryRSS) / psRSS; ratio < 0.5 || ratio > 2.0 {
				t.Errorf("RSS parity off: witr=%d bytes, ps=%.0f bytes (ratio %.2f)", me.MemoryRSS, psRSS, ratio)
			}
		}
	}

	// PID 1 is long-lived, so its lifetime-average CPU% should closely track ps's
	// (same formula, same /proc source) — a gross formula error would blow past 1%.
	if p1 := findListedPID(procs, 1); p1 != nil {
		if out, err := exec.Command("ps", "-p", "1", "-o", "%cpu=").Output(); err == nil {
			if psCPU, perr := strconv.ParseFloat(strings.TrimSpace(string(out)), 64); perr == nil {
				if diff := p1.CPUPercent - psCPU; diff > 1.0 || diff < -1.0 {
					t.Errorf("PID 1 CPU%% parity: witr=%.3f, ps=%.3f", p1.CPUPercent, psCPU)
				}
			}
		}
	}
}
