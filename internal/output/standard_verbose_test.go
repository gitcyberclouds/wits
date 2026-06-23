package output

import (
	"bytes"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/pranshuparmar/witr/pkg/model"
)

// richVerboseResult populates every optional field so a verbose render lights
// up all of its sections (resource/memory/IO/files/FDs/socket/threads/children).
func richVerboseResult() model.Result {
	proc := model.Process{
		PID:       1234,
		PPID:      1,
		Command:   "nginx",
		Cmdline:   "/usr/sbin/nginx -g daemon off;",
		User:      "nginx",
		StartedAt: time.Date(2024, 1, 1, 10, 0, 0, 0, time.UTC),
		Health:    "healthy",
		Forked:    "not-forked",
		Memory: model.MemoryInfo{
			VMS: 200 * 1024 * 1024, RSS: 50 * 1024 * 1024,
			VMSMB: 200, RSSMB: 50, Shared: 10 * 1024 * 1024,
		},
		IO:          model.IOStats{ReadBytes: 5 * 1024 * 1024, WriteBytes: 2 * 1024 * 1024, ReadOps: 100, WriteOps: 50},
		FDCount:     3,
		FDLimit:     1024,
		FileDescs:   []string{"0 -> /dev/null", "1 -> /var/log/nginx.log", "2 -> /etc/nginx.conf"},
		ThreadCount: 4,
	}
	return model.Result{
		Target:          model.Target{Type: model.TargetName, Value: "nginx"},
		Process:         proc,
		Ancestry:        []model.Process{{PID: 1, Command: "systemd"}, proc},
		Source:          model.Source{Type: model.SourceSystemd, Name: "nginx.service"},
		Warnings:        []string{"Process is running as root"},
		ResourceContext: &model.ResourceContext{CPUUsage: 85.0, MemoryUsage: 40 * 1024 * 1024, PreventsSleep: true, ThermalState: "Heavy"},
		FileContext:     &model.FileContext{OpenFiles: 90, FileLimit: 100, LockedFiles: []string{"/var/run/a.lock", "/var/run/b.lock"}},
		SocketInfo:      &model.SocketInfo{State: "TIME_WAIT", Explanation: "waiting for delayed packets", Workaround: "use SO_REUSEADDR"},
		Children:        []model.Process{{PID: 2000, Command: "worker"}},
	}
}

func TestRenderStandardColoredVerbose(t *testing.T) {
	var buf bytes.Buffer
	RenderStandard(&buf, richVerboseResult(), true, true)
	out := buf.String()

	for _, want := range []string{
		"CPU", "Energy", "Thermal", "Memory", "Virtual", "Resident", "Shared",
		"I/O Statistics", "Open Files", "Locks", "File Descriptors", "/var/log/nginx.log",
		"Socket", "waiting for delayed packets", "Threads", "Children", "worker",
		"Warnings", "running as root",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("colored+verbose output missing %q\n---\n%s", want, out)
		}
	}
}

func TestRenderStandardPlainVerbose(t *testing.T) {
	var buf bytes.Buffer
	RenderStandard(&buf, richVerboseResult(), false, true)
	out := buf.String()

	for _, want := range []string{
		"CPU         :", "Memory:", "I/O Statistics", "Open Files  :",
		"File Descriptors:", "Socket      :", "Threads: 4", "Children of nginx",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("plain+verbose output missing %q\n---\n%s", want, out)
		}
	}
}

func TestRenderStandardVerboseAltBranches(t *testing.T) {
	r := richVerboseResult()
	proc := r.Ancestry[len(r.Ancestry)-1]
	proc.FDLimit = 0 // unlimited descriptor limit
	proc.FileDescs = make([]string, 15)
	for i := range proc.FileDescs {
		proc.FileDescs[i] = fmt.Sprintf("%d -> /tmp/f%d", i, i)
	}
	r.Ancestry[len(r.Ancestry)-1] = proc
	r.Process = proc
	r.ResourceContext.CPUUsage = 10.0 // below the high-CPU threshold (green branch)
	r.FileContext.FileLimit = 0       // "of unlimited" open-files branch

	var buf bytes.Buffer
	RenderStandard(&buf, r, true, true)
	out := buf.String()

	for _, want := range []string{"unlimited", "Showing first", "and 5 more"} {
		if !strings.Contains(out, want) {
			t.Errorf("alt-branch verbose output missing %q\n---\n%s", want, out)
		}
	}
}
