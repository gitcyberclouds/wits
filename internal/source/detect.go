package source

import (
	"time"

	"github.com/pranshuparmar/witr/pkg/model"
)

func Detect(ancestry []model.Process) model.Source {
	// Prefer supervisor over systemd if both are present
	if src := detectContainer(ancestry); src != nil {
		return *src
	}
	if src := detectSupervisor(ancestry); src != nil {
		return *src
	}
	if src := detectSystemd(ancestry); src != nil {
		return *src
	}
	if src := detectCron(ancestry); src != nil {
		return *src
	}
	if src := detectShell(ancestry); src != nil {
		return *src
	}

	return model.Source{
		Type:       model.SourceUnknown,
		Confidence: 0.2,
	}
}

func Warnings(p []model.Process) []string {
	var w []string

	last := p[len(p)-1]

	if IsPublicBind(last.BindAddresses) {
		w = append(w, "Process is listening on a public interface")
	}

	if last.User == "root" {
		w = append(w, "Process is running as root")
	}

	if Detect(p).Type == model.SourceUnknown {
		w = append(w, "No known supervisor or service manager detected")
	}

	// Warn if process has been restarted multiple times (simple: if ancestry has >2 entries with same command)
	cmdCount := make(map[string]int)
	for _, proc := range p {
		cmdCount[proc.Command]++
	}
	for cmd, count := range cmdCount {
		if count > 2 {
			w = append(w, "Process or ancestor restarted multiple times: "+cmd)
		}
	}

	// Warn if process is very old (>90 days)
	if time.Since(last.StartedAt).Hours() > 90*24 {
		w = append(w, "Process has been running for over 90 days")
	}

	// Warn if working dir is suspicious
	suspiciousDirs := map[string]bool{"/": true, "/tmp": true, "/var/tmp": true}
	if suspiciousDirs[last.WorkingDir] {
		w = append(w, "Process is running from a suspicious working directory: "+last.WorkingDir)
	}

	// Warn if container and no healthcheck (placeholder, as healthcheck not detected)
	if last.Container != "" {
		w = append(w, "No healthcheck detected for container (best effort)")
	}

	// Warn if service name and process name mismatch
	if last.Service != "" && last.Command != "" && last.Service != last.Command {
		w = append(w, "Service name and process name do not match")
	}

	return w
}
