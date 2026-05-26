//go:build linux

package proc

import "testing"

func TestProcessState(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		fields []string
		want   string
	}{
		{name: "running", fields: []string{"R", "1", "1", "0"}, want: "R"},
		{name: "sleeping", fields: []string{"S", "1", "1"}, want: "S"},
		{name: "zombie", fields: []string{"Z"}, want: "Z"},
		{name: "stopped", fields: []string{"T"}, want: "T"},
		// /proc/<pid>/stat sometimes emits trailing flag letters; only the
		// first character is the canonical state code.
		{name: "extra characters trimmed", fields: []string{"Sl", "1"}, want: "S"},
		{name: "empty fields", fields: nil, want: ""},
		{name: "blank state token", fields: []string{""}, want: ""},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := processState(tt.fields); got != tt.want {
				t.Errorf("processState(%v) = %q, want %q", tt.fields, got, tt.want)
			}
		})
	}
}

func TestExtractContainerID(t *testing.T) {
	t.Parallel()

	const longID = "a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2"

	tests := []struct {
		name        string
		cgroup      string
		dashPrefix  string
		slashPrefix string
		want        string
	}{
		{
			name:        "docker dash-prefix scope (typical systemd-managed docker)",
			cgroup:      "0::/system.slice/docker-" + longID + ".scope",
			dashPrefix:  "docker-",
			slashPrefix: "docker/",
			want:        longID,
		},
		{
			name:        "docker slash-prefix (cgroupv1 / older docker)",
			cgroup:      "12:devices:/docker/" + longID,
			dashPrefix:  "docker-",
			slashPrefix: "docker/",
			want:        longID,
		},
		{
			name:        "podman libpod-prefix scope",
			cgroup:      "0::/user.slice/user-1000.slice/user@1000.service/app.slice/libpod-" + longID + ".scope",
			dashPrefix:  "libpod-",
			slashPrefix: "libpod/",
			want:        longID,
		},
		{
			name:        "slash-prefix path with extra ID chars uses first 64",
			cgroup:      "0::/docker/" + longID + "/extra",
			dashPrefix:  "docker-",
			slashPrefix: "docker/",
			want:        longID,
		},
		{
			name:        "slash-prefix path with too-short remainder returns empty",
			cgroup:      "0::/docker/shortid",
			dashPrefix:  "docker-",
			slashPrefix: "docker/",
			want:        "",
		},
		{
			name:        "no matching prefix",
			cgroup:      "0::/user.slice/user-1000.slice/session-2.scope",
			dashPrefix:  "docker-",
			slashPrefix: "docker/",
			want:        "",
		},
		{
			name:        "empty cgroup",
			cgroup:      "",
			dashPrefix:  "docker-",
			slashPrefix: "docker/",
			want:        "",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := extractContainerID(tt.cgroup, tt.dashPrefix, tt.slashPrefix); got != tt.want {
				t.Errorf("extractContainerID(%q, %q, %q) = %q, want %q",
					tt.cgroup, tt.dashPrefix, tt.slashPrefix, got, tt.want)
			}
		})
	}
}

func TestExtractLXCBasedContainerName(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name   string
		cgroup string
		want   string
	}{
		{
			name:   "incus user-owned container",
			cgroup: "0::/lxc.payload.user-1000_alpine-container/.lxc",
			want:   "alpine-container",
		},
		{
			name:   "lxc root-owned container (no user prefix)",
			cgroup: "0::/lxc.payload.my-container/.lxc",
			want:   "my-container",
		},
		{
			name:   "lxc container with underscores in name",
			cgroup: "0::/lxc.payload.test_raw_lxc_container_underline/.lxc",
			want:   "test_raw_lxc_container_underline",
		},
		{
			name:   "non-lxc cgroup (docker)",
			cgroup: "0::/system.slice/docker-abc123.scope",
			want:   "",
		},
		{
			name:   "empty cgroup",
			cgroup: "",
			want:   "",
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := extractLXCBasedContainerName(tt.cgroup); got != tt.want {
				t.Errorf("extractLXCBasedContainerName(%q) = %q, want %q", tt.cgroup, got, tt.want)
			}
		})
	}
}
