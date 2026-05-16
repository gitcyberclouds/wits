package proc

import (
	"context"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/pranshuparmar/witr/pkg/model"
)

const runtimeQueryTimeout = 3 * time.Second

var (
	healthRe               = regexp.MustCompile(`\(([^)]+)\)\s*$`)
	dockerCreatedAtLayouts = []string{
		"2006-01-02 15:04:05 -0700 MST",
		"2006-01-02 15:04:05.999999999 -0700 MST",
		"2006-01-02 15:04:05 -0700 -0700",
		time.RFC3339Nano,
		time.RFC3339,
	}
)

func dockerLikeList(bin, runtime string) []*model.ContainerMatch {
	ctx, cancel := context.WithTimeout(context.Background(), runtimeQueryTimeout)
	defer cancel()

	// {{.Labels}} returns the full label map as comma-separated key=value
	// pairs. The {{.Label "key"}} form is Docker-specific and fails the
	// template parser on Podman, so we read all labels and pick out the
	// compose ones ourselves.
	format := strings.Join([]string{
		"{{.ID}}",
		"{{.Names}}",
		"{{.Image}}",
		"{{.Command}}",
		"{{.State}}",
		"{{.Status}}",
		"{{.CreatedAt}}",
		"{{.Networks}}",
		"{{.Mounts}}",
		"{{.Ports}}",
		"{{.Labels}}",
	}, "|")
	out, err := exec.CommandContext(ctx, bin, "ps", "--no-trunc", "--format", format).Output()
	if err != nil {
		return nil
	}

	var matches []*model.ContainerMatch
	for _, line := range strings.Split(strings.TrimSpace(string(out)), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		parts := strings.SplitN(line, "|", 11)
		if len(parts) < 11 {
			continue
		}
		labels := parseLabelString(parts[10])
		matches = append(matches, &model.ContainerMatch{
			Runtime:           runtime,
			ID:                parts[0],
			Name:              parts[1],
			Image:             parts[2],
			Command:           strings.Trim(parts[3], "\""),
			State:             parts[4],
			Status:            parts[5],
			Health:            healthFromStatus(parts[5]),
			StartedAt:         parseDockerTime(parts[6]),
			Networks:          parts[7],
			Mounts:            parts[8],
			Ports:             parts[9],
			ComposeProject:    labels["com.docker.compose.project"],
			ComposeService:    labels["com.docker.compose.service"],
			ComposeConfigFile: labels["com.docker.compose.project.config_files"],
			ComposeWorkingDir: labels["com.docker.compose.project.working_dir"],
		})
	}
	return matches
}

// parseLabelString turns "key1=val1,key2=val2" into a map. Values with embedded
// commas are uncommon for the labels we care about (paths, project names),
// so the simple split is acceptable for now.
func parseLabelString(s string) map[string]string {
	out := map[string]string{}
	if s == "" {
		return out
	}
	for _, kv := range strings.Split(s, ",") {
		kv = strings.TrimSpace(kv)
		if i := strings.Index(kv, "="); i > 0 {
			out[kv[:i]] = kv[i+1:]
		}
	}
	return out
}

// healthFromStatus pulls "healthy" / "unhealthy" / "starting" out of a status
// like "Up 4 minutes (healthy)". Returns "" when no health check is wired.
func healthFromStatus(status string) string {
	m := healthRe.FindStringSubmatch(status)
	if len(m) != 2 {
		return ""
	}
	v := strings.ToLower(strings.TrimSpace(m[1]))
	switch v {
	case "healthy", "unhealthy", "health: starting", "starting":
		return strings.TrimPrefix(v, "health: ")
	}
	return ""
}

func parseDockerTime(s string) time.Time {
	s = strings.TrimSpace(s)
	if s == "" {
		return time.Time{}
	}
	for _, layout := range dockerCreatedAtLayouts {
		if t, err := time.Parse(layout, s); err == nil {
			return t
		}
	}
	return time.Time{}
}

func dockerLikeHostPID(bin, id string) int {
	ctx, cancel := context.WithTimeout(context.Background(), runtimeQueryTimeout)
	defer cancel()
	out, err := exec.CommandContext(ctx, bin, "inspect", "-f", "{{.State.Pid}}", id).Output()
	if err != nil {
		return 0
	}
	pid, _ := strconv.Atoi(strings.TrimSpace(string(out)))
	return pid
}

func binAvailable(name string) bool {
	_, err := exec.LookPath(name)
	return err == nil
}
