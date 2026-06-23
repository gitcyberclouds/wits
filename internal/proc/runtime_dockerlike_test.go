package proc

import (
	"testing"
)

func TestParseLabelString(t *testing.T) {
	got := parseLabelString("com.docker.compose.project=web, com.docker.compose.service=api")
	if got["com.docker.compose.project"] != "web" {
		t.Errorf("project = %q, want web", got["com.docker.compose.project"])
	}
	if got["com.docker.compose.service"] != "api" {
		t.Errorf("service = %q, want api", got["com.docker.compose.service"])
	}
	if len(parseLabelString("")) != 0 {
		t.Error("empty input should yield an empty map")
	}
}

func TestHealthFromStatus(t *testing.T) {
	tests := map[string]string{
		"Up 4 minutes (healthy)":         "healthy",
		"Up 2 seconds (unhealthy)":       "unhealthy",
		"Up 1 second (health: starting)": "starting",
		"Up 5 minutes":                   "", // no health check wired
		"Exited (0) 3 minutes ago":       "", // parens not at end of status
	}
	for in, want := range tests {
		if got := healthFromStatus(in); got != want {
			t.Errorf("healthFromStatus(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestParseDockerTime(t *testing.T) {
	if !parseDockerTime("").IsZero() {
		t.Error("empty input should yield the zero time")
	}
	if !parseDockerTime("not a timestamp").IsZero() {
		t.Error("garbage input should yield the zero time")
	}
	got := parseDockerTime("2024-01-02T15:04:05Z")
	if got.IsZero() || got.Year() != 2024 {
		t.Errorf("parseDockerTime(RFC3339) = %v, want a 2024 time", got)
	}
}
