package source

import (
	"testing"

	"github.com/pranshuparmar/witr/pkg/model"
)

func TestFindEnvVar(t *testing.T) {
	ancestry := []model.Process{
		{PID: 1, Command: "systemd", Env: []string{"PATH=/usr/bin"}},
		{PID: 100, Command: "bash", Env: []string{"FOO=bar", "BAZ=qux"}},
	}

	// The target (last element) is searched first.
	if got := findEnvVar(ancestry, "FOO"); got != "bar" {
		t.Errorf("findEnvVar(FOO) = %q, want bar", got)
	}
	// A variable only the ancestor sets is still found.
	if got := findEnvVar(ancestry, "PATH"); got != "/usr/bin" {
		t.Errorf("findEnvVar(PATH) = %q, want /usr/bin", got)
	}
	// A missing variable yields "".
	if got := findEnvVar(ancestry, "MISSING"); got != "" {
		t.Errorf("findEnvVar(MISSING) = %q, want empty", got)
	}
}

func TestDetectLXCRuntime(t *testing.T) {
	tests := []struct {
		cmd  string
		want string
	}{
		{"incusd", "incus"},
		{"lxd", "lxd"},
		{"lxc-start", "lxc"},
		{"unrelated", "lxc"}, // fallback when no known manager is in the chain
	}
	for _, tt := range tests {
		got := detectLXCRuntime([]model.Process{{Command: tt.cmd}})
		if got != tt.want {
			t.Errorf("detectLXCRuntime(%q) = %q, want %q", tt.cmd, got, tt.want)
		}
	}
}
