//go:build linux

package proc

import "testing"

func TestServiceUnitFromCgroup(t *testing.T) {
	tests := []struct {
		name    string
		content string
		want    string
	}{
		{"v2 system service", "0::/system.slice/nginx.service", "nginx.service"},
		{"v2 init scope", "0::/init.scope", ""},
		{"v2 login session scope", "0::/user.slice/user-1000.slice/session-2.scope", ""},
		{"scope nested under user manager", "0::/user.slice/user-1000.slice/user@1000.service/app.slice/foo.scope", ""},
		{"user service leaf", "0::/user.slice/user-1000.slice/user@1000.service/app.slice/myapp.service", "myapp.service"},
		{"delegated sub-cgroups under a service", "0::/system.slice/myapp.service/sub/leaf", "myapp.service"},
		{"cgroup v1 systemd hierarchy", "1:name=systemd:/system.slice/sshd.service\n4:cpu,cpuacct:/system.slice/sshd.service", "sshd.service"},
		{"cgroup v1 non-systemd skipped", "4:cpu,cpuacct:/system.slice/nginx.service", ""},
		{"root only", "0::/", ""},
		{"trailing newline tolerated", "0::/system.slice/redis.service\n", "redis.service"},
		{"empty", "", ""},
	}
	for _, tt := range tests {
		if got := serviceUnitFromCgroup(tt.content); got != tt.want {
			t.Errorf("%s: serviceUnitFromCgroup(%q) = %q, want %q", tt.name, tt.content, got, tt.want)
		}
	}
}
