package source

import (
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/pranshuparmar/witr/pkg/model"
)

func TestDetectShellWindowsCaseInsensitive(t *testing.T) {
	// Windows reports the shell with mixed casing (e.g. "Explorer.EXE"). Shell
	// detection must be case-insensitive, otherwise desktop-launched apps fall
	// through to SourceUnknown and pick up a spurious no-supervisor warning.
	for _, shell := range []string{"Explorer.EXE", "cmd.EXE", "PowerShell.exe"} {
		ancestry := []model.Process{
			{PID: 100, Command: shell},
			{PID: 200, PPID: 100, Command: "Claude.exe"},
		}
		if got := Detect(ancestry).Type; got != model.SourceShell {
			t.Errorf("Detect with %q ancestor = %v; want SourceShell", shell, got)
		}
		if slices.Contains(Warnings(ancestry, 0), "No known supervisor or service manager detected") {
			t.Errorf("%q ancestor should not raise the no-supervisor warning", shell)
		}
	}
}

func TestDetectWindowsSystemKernel(t *testing.T) {
	// A process rooted at the Windows System process (PID 4) is a kernel/system
	// process, not an unsupervised one. It must resolve to an init source and
	// must not raise the no-supervisor warning (the Memory Compression false
	// positive).
	ancestry := []model.Process{
		{PID: 4, Command: "System"},
		{PID: 4108, PPID: 4, Command: "Memory Compression"},
	}
	if got := Detect(ancestry).Type; got != model.SourceInit {
		t.Errorf("Detect with System (pid 4) root = %v; want SourceInit", got)
	}
	if slices.Contains(Warnings(ancestry, 0), "No known supervisor or service manager detected") {
		t.Errorf("System-rooted process should not raise the no-supervisor warning")
	}
}

func TestWarningsDetectsLDPreload(t *testing.T) {
	p := []model.Process{
		{PID: 999999, Command: "pm2", Cmdline: "pm2"},
		{
			PID:        123,
			Command:    "bash",
			StartedAt:  time.Now(),
			User:       "bob",
			WorkingDir: "/home/bob",
			Env:        []string{"LD_PRELOAD=/tmp/libhack.so"},
		},
	}

	warnings := Warnings(p, 0)
	if !slices.Contains(warnings, "Process sets LD_PRELOAD (potential library injection)") {
		t.Fatalf("expected LD_PRELOAD warning, got: %v", warnings)
	}
}

func TestWarningsDetectsDYLDVars(t *testing.T) {
	p := []model.Process{
		{PID: 999999, Command: "pm2", Cmdline: "pm2"},
		{
			PID:        123,
			Command:    "zsh",
			StartedAt:  time.Now(),
			User:       "bob",
			WorkingDir: "/home/bob",
			Env: []string{
				"DYLD_LIBRARY_PATH=/tmp",
				"DYLD_INSERT_LIBRARIES=/tmp/inject.dylib",
			},
		},
	}

	warnings := Warnings(p, 0)
	want := "Process sets DYLD_* variables (potential library injection): DYLD_INSERT_LIBRARIES, DYLD_LIBRARY_PATH"
	if !slices.Contains(warnings, want) {
		t.Fatalf("expected DYLD warning %q, got: %v", want, warnings)
	}
}

func TestWarningsIgnoresEmptyPreloadVars(t *testing.T) {
	p := []model.Process{
		{PID: 999999, Command: "pm2", Cmdline: "pm2"},
		{
			PID:        123,
			Command:    "zsh",
			StartedAt:  time.Now(),
			User:       "bob",
			WorkingDir: "/home/bob",
			Env: []string{
				"LD_PRELOAD=",
				"DYLD_INSERT_LIBRARIES=",
			},
		},
	}

	warnings := Warnings(p, 0)
	if slices.Contains(warnings, "Process sets LD_PRELOAD (potential library injection)") {
		t.Fatalf("did not expect LD_PRELOAD warning, got: %v", warnings)
	}
	if slices.Contains(warnings, "Process sets DYLD_* variables (potential library injection): DYLD_INSERT_LIBRARIES") {
		t.Fatalf("did not expect DYLD warning, got: %v", warnings)
	}
}

// checks if the order of env vars warnings are deterministic
func FuzzEnvSuspiciousWarningsDeterministic(f *testing.F) {
	f.Add("LD_PRELOAD=/tmp/lib.so")
	f.Add("DYLD_LIBRARY_PATH=/tmp\nDYLD_INSERT_LIBRARIES=/tmp/inject.dylib")
	f.Add("DYLD_LIBRARY_PATH=\nLD_PRELOAD=")
	f.Add("")

	f.Fuzz(func(t *testing.T, input string) {
		parts := strings.Split(input, "\n")
		if len(parts) > 50 {
			parts = parts[:50]
		}
		for i := range parts {
			if len(parts[i]) > 200 {
				parts[i] = parts[i][:200]
			}
		}

		w1 := envSuspiciousWarnings(parts)
		w2 := envSuspiciousWarnings(parts)
		if !slices.Equal(w1, w2) {
			t.Fatalf("expected deterministic output, got %v vs %v", w1, w2)
		}
	})
}

func TestEnvSuspiciousWarnings(t *testing.T) {
	tests := []struct {
		name string
		env  []string
		want []string
	}{
		{
			name: "LD_PRELOAD",
			env:  []string{"LD_PRELOAD=/tmp/libhack.so"},
			want: []string{"Process sets LD_PRELOAD (potential library injection)"},
		},
		{
			name: "DYLD keys sorted and deduped",
			env: []string{
				"DYLD_LIBRARY_PATH=/tmp",
				"DYLD_INSERT_LIBRARIES=/tmp/inject.dylib",
				"DYLD_LIBRARY_PATH=/tmp", // dup
			},
			want: []string{
				"Process sets DYLD_* variables (potential library injection): DYLD_INSERT_LIBRARIES, DYLD_LIBRARY_PATH",
			},
		},
		{
			name: "ignores empty values (current behavior)",
			env:  []string{"LD_PRELOAD=", "DYLD_INSERT_LIBRARIES="},
			want: nil,
		},
		{
			name: "value with '=' still counts",
			env:  []string{"LD_PRELOAD=a=b"},
			want: []string{"Process sets LD_PRELOAD (potential library injection)"},
		},
		{
			name: "no '=' ignored",
			env:  []string{"LD_PRELOAD"},
			want: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := envSuspiciousWarnings(tt.env)
			if !slices.Equal(got, tt.want) {
				t.Fatalf("got %v, want %v", got, tt.want)
			}
		})
	}
}

// attempts to cause a panic when checking for env vars
func FuzzWarningsNoPanic(f *testing.F) {
	f.Add("LD_PRELOAD=/tmp/lib.so")
	f.Add("DYLD_INSERT_LIBRARIES=/tmp/inject.dylib")
	f.Add("NOT_AN_ENV")

	f.Fuzz(func(t *testing.T, entry string) {
		if len(entry) > 2000 {
			entry = entry[:2000]
		}
		p := []model.Process{
			{
				PID:        123,
				Command:    "test",
				Cmdline:    "test",
				StartedAt:  time.Now(),
				User:       "bob",
				WorkingDir: "/home/bob",
				Env:        []string{entry},
			},
		}

		_ = Warnings(p, 0)
	})
}

func TestWarningsDetectsDeletedExecutable(t *testing.T) {
	p := []model.Process{
		{
			PID:        123,
			Command:    "nginx",
			StartedAt:  time.Now(),
			ExeDeleted: true,
		},
	}

	warnings := Warnings(p, 0)
	want := "Process is running from a deleted binary (potential library injection or pending update)"
	if !slices.Contains(warnings, want) {
		t.Fatalf("expected deleted binary warning, got: %v", warnings)
	}
}

func TestEnrichSocketInfo(t *testing.T) {
	tests := []struct {
		state           string
		wantExplanation string
		wantWorkaround  string // empty means no workaround should be set
	}{
		{
			state:           "TIME_WAIT",
			wantExplanation: "The local OS is holding the port in a protocol-wait state to ensure all packets are received.",
			wantWorkaround:  "Wait ~60s for the OS to release it, or enable SO_REUSEADDR in your code.",
		},
		{
			state:           "CLOSE_WAIT",
			wantExplanation: "The remote end has closed the connection, but the local application hasn't responded.",
			wantWorkaround:  "This usually indicates a resource leak in the application. Restart the process.",
		},
		{
			state:           "LISTEN",
			wantExplanation: "The process is actively waiting for incoming connections.",
			wantWorkaround:  "", // no workaround for a healthy listener
		},
		{
			state:           "ESTABLISHED",
			wantExplanation: "The connection is active and data can be transferred.",
			wantWorkaround:  "",
		},
		{
			state:           "FIN_WAIT_1",
			wantExplanation: "The connection is in the process of being closed.",
			wantWorkaround:  "",
		},
		{
			state:           "FIN_WAIT_2",
			wantExplanation: "The connection is in the process of being closed.",
			wantWorkaround:  "",
		},
	}

	for _, tt := range tests {
		si := &model.SocketInfo{State: tt.state}
		EnrichSocketInfo(si)
		if si.Explanation != tt.wantExplanation {
			t.Errorf("state %s: got explanation %q, want %q", tt.state, si.Explanation, tt.wantExplanation)
		}
		if si.Workaround != tt.wantWorkaround {
			t.Errorf("state %s: got workaround %q, want %q", tt.state, si.Workaround, tt.wantWorkaround)
		}
	}
}

// TestEnrichSocketInfoUnknownState ensures unknown states pass through
// without panic and without populating either string. Forward-compatible
// states added by newer kernels (e.g. NEW_SYN_RECV) must not corrupt the
// record.
func TestEnrichSocketInfoUnknownState(t *testing.T) {
	si := &model.SocketInfo{State: "MYSTERY"}
	EnrichSocketInfo(si)
	if si.Explanation != "" || si.Workaround != "" {
		t.Errorf("unknown state should leave fields blank, got explanation=%q workaround=%q",
			si.Explanation, si.Workaround)
	}
}

// TestEnrichSocketInfoNilSafe ensures EnrichSocketInfo is safe to call on a
// nil pointer (some callers pass *SocketInfo conditionally).
func TestEnrichSocketInfoNilSafe(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("EnrichSocketInfo(nil) panicked: %v", r)
		}
	}()
	EnrichSocketInfo(nil)
}
