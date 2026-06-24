package source

import (
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/pranshuparmar/witr/pkg/model"
)

// baseProc returns a benign process record that produces no warnings by
// itself. Tests mutate one field at a time to exercise individual rules.
func baseProc() model.Process {
	return model.Process{
		PID:        1234,
		Command:    "nginx",
		User:       "www-data",
		StartedAt:  time.Now().Add(-1 * time.Hour),
		WorkingDir: "/var/www",
		Health:     "healthy",
	}
}

// wrap forces Warnings to use a known source type so we don't depend on
// the platform-specific Detect() and its real-system probing.
func wrap(p model.Process) []string {
	parent := baseProc()
	parent.PID = 1
	parent.Command = "systemd"
	return Warnings([]model.Process{parent, p}, 0, model.SourceSystemd)
}

func contains(haystack []string, needle string) bool {
	for _, h := range haystack {
		if strings.Contains(h, needle) {
			return true
		}
	}
	return false
}

func TestWarningsHealthStates(t *testing.T) {
	t.Parallel()

	tests := []struct {
		health string
		want   string
	}{
		{"zombie", "zombie"},
		{"stopped", "stopped"},
		{"high-cpu", "high CPU"},
		{"high-mem", "high memory"},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.health, func(t *testing.T) {
			t.Parallel()
			p := baseProc()
			p.Health = tt.health
			if !contains(wrap(p), tt.want) {
				t.Errorf("expected warning containing %q for health=%q, got: %v",
					tt.want, tt.health, wrap(p))
			}
		})
	}
}

func TestWarningsPublicBind(t *testing.T) {
	t.Parallel()

	p := baseProc()
	p.Sockets = []model.Socket{{Address: "0.0.0.0", Port: 443, State: "LISTEN"}}
	if !contains(wrap(p), "public interface") {
		t.Errorf("expected public-interface warning, got: %v", wrap(p))
	}
}

func TestWarningsRootUser(t *testing.T) {
	t.Parallel()

	p := baseProc()
	p.User = "root"
	if !contains(wrap(p), "running as root") {
		t.Errorf("expected root warning, got: %v", wrap(p))
	}
}

func TestWarningsDangerousCapabilities(t *testing.T) {
	t.Parallel()

	p := baseProc()
	p.Capabilities = []string{"CAP_NET_BIND_SERVICE", "CAP_SYS_PTRACE", "CAP_SYS_ADMIN"}
	w := wrap(p)

	if !contains(w, "dangerous capabilities") {
		t.Fatalf("expected dangerous-capabilities warning, got: %v", w)
	}
	// Only the dangerous ones should appear in the message.
	if !contains(w, "CAP_SYS_PTRACE") || !contains(w, "CAP_SYS_ADMIN") {
		t.Errorf("expected dangerous caps listed in warning, got: %v", w)
	}
	if contains(w, "CAP_NET_BIND_SERVICE") {
		t.Errorf("benign cap should not appear in warning, got: %v", w)
	}
}

func TestWarningsBenignCapabilitiesSuppressed(t *testing.T) {
	t.Parallel()

	p := baseProc()
	p.Capabilities = []string{"CAP_NET_BIND_SERVICE"}
	if contains(wrap(p), "dangerous capabilities") {
		t.Errorf("benign capability triggered warning, got: %v", wrap(p))
	}
}

func TestWarningsRootSuppressesCapabilitiesCheck(t *testing.T) {
	t.Parallel()

	// Per the implementation, root already triggers its own warning and the
	// capability check is skipped (root has them all anyway).
	p := baseProc()
	p.User = "root"
	p.Capabilities = []string{"CAP_SYS_ADMIN"}
	w := wrap(p)
	if contains(w, "dangerous capabilities") {
		t.Errorf("dangerous-capabilities warning should be suppressed for root, got: %v", w)
	}
	if !contains(w, "running as root") {
		t.Errorf("root warning still expected, got: %v", w)
	}
}

func TestWarningsUnknownSupervisor(t *testing.T) {
	t.Parallel()

	p := baseProc()
	got := Warnings([]model.Process{p}, 0, model.SourceUnknown)
	hasWarning := contains(got, "No known supervisor")
	if runtime.GOOS == "windows" {
		// Suppressed on Windows: ancestry truncates at orphaned processes, so
		// "unknown source" is normal rather than a sign of no supervision.
		if hasWarning {
			t.Errorf("unknown-supervisor warning should be suppressed on Windows, got: %v", got)
		}
		return
	}
	if !hasWarning {
		t.Errorf("expected unknown-supervisor warning, got: %v", got)
	}
}

func TestWarningsLongRunning(t *testing.T) {
	t.Parallel()

	p := baseProc()
	p.StartedAt = time.Now().Add(-100 * 24 * time.Hour)
	if !contains(wrap(p), "over 90 days") {
		t.Errorf("expected long-running warning, got: %v", wrap(p))
	}
}

// A zero start time means we couldn't read it (e.g. a protected Windows
// process), not that the process is ancient — it must not trigger the
// long-running warning. Regression guard for issue #205.
func TestWarningsZeroStartTimeNotLongRunning(t *testing.T) {
	t.Parallel()

	p := baseProc()
	p.StartedAt = time.Time{}
	if contains(wrap(p), "over 90 days") {
		t.Errorf("zero start time should not trigger long-running warning, got: %v", wrap(p))
	}
}

func TestWarningsSuspiciousWorkingDirs(t *testing.T) {
	t.Parallel()

	for _, dir := range []string{"/", "/tmp", "/var/tmp"} {
		dir := dir
		t.Run(dir, func(t *testing.T) {
			t.Parallel()
			p := baseProc()
			p.WorkingDir = dir
			if !contains(wrap(p), "suspicious working directory") {
				t.Errorf("expected suspicious-dir warning for %q, got: %v", dir, wrap(p))
			}
		})
	}
}

func TestWarningsContainerHealthcheck(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		healthcheck string
		wantWarning bool
	}{
		{"absent warns", "absent", true},
		{"present does not warn", "present", false},
		{"unknown does not warn", "", false},
	}
	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			p := baseProc()
			p.Container = "docker (abc123)"
			p.ContainerHealthcheck = tc.healthcheck
			if got := contains(wrap(p), "healthcheck"); got != tc.wantWarning {
				t.Errorf("ContainerHealthcheck=%q: warning=%v, want %v (%v)", tc.healthcheck, got, tc.wantWarning, wrap(p))
			}
		})
	}
}

func TestWarningsSnapAndFlatpakSkipHealthcheck(t *testing.T) {
	t.Parallel()

	// Snap and Flatpak don't use healthchecks — they shouldn't trigger this
	// warning even though the Container field is set.
	for _, container := range []string{"snap: discord", "flatpak: org.signal.Signal"} {
		container := container
		t.Run(container, func(t *testing.T) {
			t.Parallel()
			p := baseProc()
			p.Container = container
			if contains(wrap(p), "healthcheck") {
				t.Errorf("healthcheck warning should not fire for %q, got: %v", container, wrap(p))
			}
		})
	}
}

func TestWarningsServiceNameMismatch(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		command     string
		service     string
		wantWarning bool
	}{
		{"match", "nginx", "nginx.service", false},
		{"systemd suffix tolerated", "postgres", "postgresql.service", false}, // svcCore contains cmd substring
		{"systemd template instance tolerated", "agetty", "getty@tty1.service", false},
		{"mismatch", "nginx", "redis.service", true},
		{"empty service skipped", "nginx", "", false},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			p := baseProc()
			p.Command = tt.command
			p.Service = tt.service
			got := contains(wrap(p), "Service name and process name")
			if got != tt.wantWarning {
				t.Errorf("service mismatch warning for cmd=%q svc=%q: got=%v want=%v\n%v",
					tt.command, tt.service, got, tt.wantWarning, wrap(p))
			}
		})
	}
}

func TestWarningsRestartCount(t *testing.T) {
	t.Parallel()

	chain := []model.Process{{PID: 1, Command: "systemd"}, baseProc()}

	// The warning is driven by the real restart count (e.g. systemd NRestarts),
	// not by the shape of the ancestry.
	if got := Warnings(chain, 7, model.SourceSystemd); !contains(got, "restarted 7 times") {
		t.Errorf("expected restart warning for 7 restarts, got: %v", got)
	}
	if got := Warnings(chain, 5, model.SourceSystemd); contains(got, "restarted") {
		t.Errorf("no restart warning expected for 5 restarts, got: %v", got)
	}
}

func TestWarningsBenignProcProducesNoSpurious(t *testing.T) {
	t.Parallel()

	got := wrap(baseProc())
	for _, w := range got {
		// The base fixture is intentionally benign; any warning here is a
		// false positive that should be investigated.
		t.Errorf("benign fixture produced warning: %q", w)
	}
}

func TestWarningsEmptyInputReturnsNil(t *testing.T) {
	t.Parallel()

	if got := Warnings(nil, 0); got != nil {
		t.Errorf("Warnings(nil) = %v, want nil", got)
	}
}
