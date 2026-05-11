package proc

import (
	"os/exec"
	"strings"
)

func resolveDockerProxyContainer(cmdline string) string {
	var containerIP string
	parts := strings.Fields(cmdline)
	for i, part := range parts {
		if part == "-container-ip" && i+1 < len(parts) {
			containerIP = parts[i+1]
			break
		}
	}
	if containerIP == "" {
		return ""
	}

	out, err := exec.Command("docker", "network", "inspect", "bridge",
		"--format", "{{range .Containers}}{{.Name}}:{{.IPv4Address}}{{\"\\n\"}}{{end}}").Output()
	if err != nil {
		return ""
	}

	for _, line := range strings.Split(strings.TrimSpace(string(out)), "\n") {
		if line == "" {
			continue
		}
		colonIdx := strings.Index(line, ":")
		if colonIdx == -1 {
			continue
		}
		name := line[:colonIdx]
		ip := strings.Split(line[colonIdx+1:], "/")[0]
		if ip == containerIP {
			return "target: " + name
		}
	}
	return ""
}
