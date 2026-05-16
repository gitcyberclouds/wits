package output

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"github.com/pranshuparmar/witr/pkg/model"
)

func TestRenderContainerFallback(t *testing.T) {
	match := &model.ContainerMatch{
		Runtime: "docker",
		ID:      "abc123",
		Name:    "my-container",
		Image:   "nginx:latest",
		Ports:   "0.0.0.0:8080->80/tcp",
	}

	var buf bytes.Buffer
	RenderContainerFallback(&buf, "port 8080", match, false, false)
	out := buf.String()

	expected := []string{
		"Target      : port 8080",
		"Container   : my-container (id abc123)",
		"Image       : nginx:latest",
		"Sockets     : 0.0.0.0:8080->80/tcp",
		"Why It Exists",
		"Source      : docker",
		"Note",
	}
	for _, want := range expected {
		if !strings.Contains(out, want) {
			t.Errorf("RenderContainerFallback output missing %q\nGot:\n%s", want, out)
		}
	}
}

func TestRenderContainerFallbackWithCompose(t *testing.T) {
	match := &model.ContainerMatch{
		Runtime:        "docker",
		ID:             "abc123",
		Name:           "myapp-db-1",
		Image:          "postgres:16",
		Ports:          "0.0.0.0:5432->5432/tcp",
		ComposeProject: "myapp",
		ComposeService: "db",
	}

	var buf bytes.Buffer
	RenderContainerFallback(&buf, "port 5432", match, false, false)
	out := buf.String()

	if !strings.Contains(out, "docker-compose: myapp/db") {
		t.Errorf("expected compose source label, got:\n%s", out)
	}
}

func TestRenderContainerFallbackRuntimeLabel(t *testing.T) {
	match := &model.ContainerMatch{
		Runtime: "podman",
		ID:      "def456",
		Name:    "rootless",
		Image:   "alpine:3",
	}

	var buf bytes.Buffer
	RenderContainerFallback(&buf, "container rootless", match, false, false)
	out := buf.String()

	if !strings.Contains(out, "Source      : podman") {
		t.Errorf("expected podman source label, got:\n%s", out)
	}
}

func TestRenderContainerFallbackShort(t *testing.T) {
	match := &model.ContainerMatch{
		Runtime: "docker",
		ID:      "abc123",
		Name:    "my-container",
		Image:   "nginx:latest",
		Ports:   "0.0.0.0:8080->80/tcp",
	}

	var buf bytes.Buffer
	RenderContainerFallbackShort(&buf, "port 8080", match, false)
	out := buf.String()

	want := "docker → my-container"
	if !strings.Contains(out, want) {
		t.Errorf("short output missing %q, got: %s", want, out)
	}
	if strings.Count(out, "\n") != 1 {
		t.Errorf("short output should be single line, got: %s", out)
	}
}

func TestRenderContainerFallbackShortWithCompose(t *testing.T) {
	match := &model.ContainerMatch{
		Runtime:        "docker",
		ID:             "abc123",
		Name:           "redis",
		Image:          "redis:7-alpine",
		ComposeProject: "myapp",
		ComposeService: "redis",
	}

	var buf bytes.Buffer
	RenderContainerFallbackShort(&buf, "container redis", match, false)
	out := buf.String()

	want := "docker → myapp (docker-compose) → redis"
	if !strings.Contains(out, want) {
		t.Errorf("short output missing chain %q, got: %s", want, out)
	}
}

func TestRenderContainerFallbackTree(t *testing.T) {
	match := &model.ContainerMatch{
		Runtime:        "docker",
		ID:             "abc123",
		Name:           "redis",
		Image:          "redis:7-alpine",
		ComposeProject: "myapp",
		ComposeService: "redis",
	}

	var buf bytes.Buffer
	RenderContainerFallbackTree(&buf, match, false)
	out := buf.String()

	for _, want := range []string{"docker\n", "└─ myapp (docker-compose)", "└─ redis"} {
		if !strings.Contains(out, want) {
			t.Errorf("tree output missing %q, got:\n%s", want, out)
		}
	}
}

func TestRenderContainerFallbackWarnings(t *testing.T) {
	match := &model.ContainerMatch{
		Runtime: "docker",
		ID:      "abc123",
		Name:    "redis",
		Image:   "redis:7-alpine",
	}

	var buf bytes.Buffer
	RenderContainerFallbackWarnings(&buf, match, false)
	out := buf.String()

	for _, want := range []string{"Container   : redis", "No warnings"} {
		if !strings.Contains(out, want) {
			t.Errorf("warnings output missing %q, got: %s", want, out)
		}
	}
}

func TestContainerFallbackToJSON(t *testing.T) {
	match := &model.ContainerMatch{
		Runtime: "docker",
		ID:      "abc123",
		Name:    "sql-proxy",
		Image:   "gcr.io/cloud-sql-connectors/cloud-sql-proxy:2.13.0",
		Ports:   "127.0.0.1:5432->5432/tcp",
	}

	jsonStr, err := ContainerFallbackToJSON("port 5432", match)
	if err != nil {
		t.Fatalf("ContainerFallbackToJSON() error: %v", err)
	}

	var result map[string]interface{}
	if err := json.Unmarshal([]byte(jsonStr), &result); err != nil {
		t.Fatalf("invalid JSON output: %v", err)
	}

	if result["Target"] != "port 5432" {
		t.Errorf("Target = %v, want %q", result["Target"], "port 5432")
	}
	if result["ContainerName"] != "sql-proxy" {
		t.Errorf("ContainerName = %v, want %q", result["ContainerName"], "sql-proxy")
	}
	if result["Runtime"] != "docker" {
		t.Errorf("Runtime = %v, want %q", result["Runtime"], "docker")
	}
	if result["Source"] != "docker" {
		t.Errorf("Source = %v, want %q", result["Source"], "docker")
	}
}

func TestContainerFallbackToJSONCompose(t *testing.T) {
	match := &model.ContainerMatch{
		Runtime:        "docker",
		ID:             "def456",
		Name:           "myapp-db-1",
		Image:          "postgres:16",
		Ports:          "0.0.0.0:5432->5432/tcp",
		ComposeProject: "myapp",
		ComposeService: "db",
	}

	jsonStr, err := ContainerFallbackToJSON("port 5432", match)
	if err != nil {
		t.Fatalf("ContainerFallbackToJSON() error: %v", err)
	}

	var result map[string]interface{}
	if err := json.Unmarshal([]byte(jsonStr), &result); err != nil {
		t.Fatalf("invalid JSON output: %v", err)
	}

	if result["Source"] != "docker-compose: myapp/db" {
		t.Errorf("Source = %v, want %q", result["Source"], "docker-compose: myapp/db")
	}
}

func TestRenderContainerFallbackSanitizesOutput(t *testing.T) {
	match := &model.ContainerMatch{
		Runtime: "docker",
		ID:      "abc123",
		Name:    "evil\x1b[31mcontainer",
		Image:   "evil\x1b[0mimage",
		Ports:   "0.0.0.0:80->80/tcp",
	}

	var buf bytes.Buffer
	RenderContainerFallback(&buf, "port 80", match, false, false)
	out := buf.String()

	if strings.Contains(out, "\x1b") {
		t.Errorf("output contains raw ANSI escape sequences, sanitization failed:\n%s", out)
	}
}
