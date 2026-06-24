package source

import (
	"os"
	"strconv"
	"strings"

	"github.com/pranshuparmar/witr/pkg/model"
)

func detectContainer(ancestry []model.Process) *model.Source {
	for _, p := range ancestry {
		data, err := os.ReadFile("/proc/" + itoa(p.PID) + "/cgroup")
		if err != nil {
			continue
		}
		content := string(data)

		switch {
		case strings.Contains(content, "docker"):
			return &model.Source{
				Type: model.SourceContainer,
				Name: "docker",
			}
		case strings.Contains(content, "podman"), strings.Contains(content, "libpod"):
			return &model.Source{
				Type: model.SourceContainer,
				Name: "podman",
			}
		case strings.Contains(content, "kubepods"):
			return &model.Source{
				Type: model.SourceContainer,
				Name: "kubernetes",
			}
		case strings.Contains(content, "colima"):
			return &model.Source{
				Type: model.SourceContainer,
				Name: "colima",
			}
		case strings.Contains(content, "containerd"):
			return &model.Source{
				Type: model.SourceContainer,
				Name: "containerd",
			}
		case strings.Contains(content, "lxc.payload"):
			return &model.Source{
				Type: model.SourceContainer,
				Name: detectLXCRuntime(ancestry),
			}
		}
	}

	// Snap/Flatpak sandbox detection via environment variables
	if len(ancestry) > 0 {
		target := ancestry[len(ancestry)-1]
		for _, e := range target.Env {
			if strings.HasPrefix(e, "SNAP_NAME=") {
				return &model.Source{
					Type: model.SourceContainer,
					Name: "snap",
				}
			}
			if strings.HasPrefix(e, "FLATPAK_ID=") {
				return &model.Source{
					Type: model.SourceContainer,
					Name: "flatpak",
				}
			}
		}
	}

	return nil
}

func itoa(n int) string {
	return strconv.Itoa(n)
}

func detectLXCRuntime(ancestry []model.Process) string {
	for _, a := range ancestry {
		switch a.Command {
		case "incusd":
			return "incus"
		case "lxd":
			return "lxd"
		case "lxc-start":
			return "lxc"
		}
	}
	return "lxc" // fallback
}
