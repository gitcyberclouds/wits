package source

import "github.com/pranshuparmar/witr/pkg/model"

func DetectPrimary(chain []model.Process) string {
	for _, p := range chain {
		switch p.Command {
		case "systemd":
			return "systemd"
		case "dockerd", "containerd", "kubelet":
			return "docker"
		case "podman":
			return "podman"
		case "pm2":
			return "pm2"
		case "cron":
			return "cron"
		}
	}
	return "manual"
}
