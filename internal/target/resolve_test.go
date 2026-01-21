//go:build linux || darwin || freebsd || windows

package target

import (
	"testing"

	"github.com/pranshuparmar/witr/pkg/model"
)

func TestResolveWithExactFlag(t *testing.T) {
	tests := []struct {
		name  string
		exact bool
	}{
		{
			name:  "fuzzy matching",
			exact: false,
		},
		{
			name:  "exact matching",
			exact: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			target := model.Target{
				Type:  model.TargetName,
				Value: "test",
			}

			_, err := Resolve(target, tt.exact)

			if err == nil {
				t.Log("Resolve returned successfully (may have found matches)")
			} else {
				t.Logf("Resolve returned error: %v", err)
			}
		})
	}
}

func TestResolvePIDWithExactFlag(t *testing.T) {
	tests := []struct {
		name  string
		exact bool
		pid   string
	}{
		{"exact flag true", true, "1"},
		{"exact flag false", false, "1"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			target := model.Target{
				Type:  model.TargetPID,
				Value: tt.pid,
			}

			pids, err := Resolve(target, tt.exact)
			if err != nil {
				t.Fatalf("Resolve failed: %v", err)
			}

			if len(pids) != 1 {
				t.Fatalf("expected 1 PID, got %d", len(pids))
			}

			if pids[0] != 1 {
				t.Fatalf("expected PID 1, got %d", pids[0])
			}
		})
	}
}

func TestResolvePortWithExactFlag(t *testing.T) {
	tests := []struct {
		name  string
		exact bool
		port  string
	}{
		{"exact flag true", true, "22"},
		{"exact flag false", false, "22"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			target := model.Target{
				Type:  model.TargetPort,
				Value: tt.port,
			}

			_, err := Resolve(target, tt.exact)

			if err == nil {
				t.Log("Resolve returned successfully (may have found matches)")
			} else {
				t.Logf("Resolve returned error: %v", err)
			}
		})
	}
}
