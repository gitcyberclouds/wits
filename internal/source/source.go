package source

import "github.com/pranshuparmar/witr/pkg/model"

func DetectPrimary(chain []model.Process) string {
	for _, p := range chain {
		switch p.Command {
		case "systemd":
			return "systemd"
		case "init", "/sbin/init":
			// Check if this is a FreeBSD rc.d service
			if p.Service != "" {
				return p.Service
			}
			return "bsdrc"
		case "dockerd", "containerd", "kubelet":
			return "docker"
		case "podman":
			return "podman"
		case "pm2":
			return "pm2"
		case "cron":
			return "cron"
		}
		// Also check Service field directly
		if p.Service != "" {
			return p.Service
		}
	}
	return "manual"
}
