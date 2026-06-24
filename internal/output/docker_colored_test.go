package output

import (
	"bytes"
	"strings"
	"testing"
	"time"

	"github.com/pranshuparmar/witr/pkg/model"
)

func TestRenderContainerFallbackColoredVerbose(t *testing.T) {
	match := &model.ContainerMatch{
		Runtime:           "docker",
		ID:                "abc123def456",
		Name:              "web",
		Image:             "nginx:latest",
		Command:           "nginx -g daemon off",
		Networks:          "bridge",
		Ports:             "0.0.0.0:8080->80/tcp",
		StartedAt:         time.Now().Add(-10 * time.Minute),
		CreatedAt:         time.Now().Add(-20 * time.Minute),
		Mounts:            "/host:/container",
		ComposeConfigFile: "/app/docker-compose.yml",
		ComposeWorkingDir: "/app",
	}

	var buf bytes.Buffer
	RenderContainerFallback(&buf, "port 8080", match, true, true) // colored + verbose
	out := buf.String()

	for _, want := range []string{
		"web", "nginx:latest", "nginx -g daemon", "Started", "Created",
		"Network", "bridge", "Why It Exists", "Source", "Mounts", "/host:/container",
		"Compose File", "Compose Dir", "Note",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("colored+verbose container fallback missing %q\n---\n%s", want, out)
		}
	}
}
