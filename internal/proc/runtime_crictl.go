package proc

import (
	"context"
	"encoding/json"
	"os/exec"
	"strings"
	"time"

	"github.com/pranshuparmar/witr/pkg/model"
)

func init() { registerRuntime(crictlRuntime{}) }

type crictlRuntime struct{}

func (crictlRuntime) Name() string    { return "k8s" }
func (crictlRuntime) Available() bool { return binAvailable("crictl") }

func (crictlRuntime) List() []*model.ContainerMatch {
	ctx, cancel := context.WithTimeout(context.Background(), runtimeQueryTimeout)
	defer cancel()
	out, err := exec.CommandContext(ctx, "crictl", "ps", "-o", "json").Output()
	if err != nil {
		return nil
	}

	var payload struct {
		Containers []struct {
			ID        string                 `json:"id"`
			Image     struct{ Image string } `json:"image"`
			ImageRef  string                 `json:"imageRef"`
			Metadata  struct{ Name string }  `json:"metadata"`
			Labels    map[string]string      `json:"labels"`
			State     string                 `json:"state"`
			CreatedAt string                 `json:"createdAt"`
		} `json:"containers"`
	}
	if err := json.Unmarshal(out, &payload); err != nil {
		return nil
	}

	var matches []*model.ContainerMatch
	for _, c := range payload.Containers {
		started, _ := time.Parse(time.RFC3339Nano, c.CreatedAt)
		matches = append(matches, &model.ContainerMatch{
			Runtime:   "k8s",
			ID:        c.ID,
			Name:      c.Metadata.Name,
			Image:     c.Image.Image,
			State:     strings.TrimPrefix(c.State, "CONTAINER_"),
			Status:    strings.TrimPrefix(c.State, "CONTAINER_"),
			StartedAt: started,
		})
	}
	return matches
}

func (crictlRuntime) HostPID(id string) int {
	info, _ := crictlInspect(id)
	return info.Info.Pid
}

// Enrich populates Command, Mounts, and a more precise StartedAt by calling
// `crictl inspect` for the resolved container. Skips fields the inspect
// payload doesn't carry; partial enrichment is fine.
func (crictlRuntime) Enrich(match *model.ContainerMatch) {
	payload, ok := crictlInspect(match.ID)
	if !ok {
		return
	}
	if args := payload.Info.RuntimeSpec.Process.Args; len(args) > 0 {
		match.Command = strings.Join(args, " ")
	}
	if len(payload.Status.Mounts) > 0 {
		var parts []string
		for _, m := range payload.Status.Mounts {
			entry := m.HostPath + " → " + m.ContainerPath
			if m.Readonly {
				entry += " (ro)"
			}
			parts = append(parts, entry)
		}
		match.Mounts = strings.Join(parts, ", ")
	}
	if payload.Status.StartedAt != "" {
		if t, err := time.Parse(time.RFC3339Nano, payload.Status.StartedAt); err == nil {
			match.StartedAt = t
		}
	}
}

type crictlInspectPayload struct {
	Status struct {
		StartedAt string `json:"startedAt"`
		Mounts    []struct {
			ContainerPath string `json:"containerPath"`
			HostPath      string `json:"hostPath"`
			Readonly      bool   `json:"readonly"`
		} `json:"mounts"`
	} `json:"status"`
	Info struct {
		Pid         int `json:"pid"`
		RuntimeSpec struct {
			Process struct {
				Args []string `json:"args"`
			} `json:"process"`
		} `json:"runtimeSpec"`
	} `json:"info"`
}

func crictlInspect(id string) (crictlInspectPayload, bool) {
	var p crictlInspectPayload
	ctx, cancel := context.WithTimeout(context.Background(), runtimeQueryTimeout)
	defer cancel()
	out, err := exec.CommandContext(ctx, "crictl", "inspect", id).Output()
	if err != nil {
		return p, false
	}
	if err := json.Unmarshal(out, &p); err == nil {
		return p, true
	}
	// Older crictl versions wrap the `info` field as a JSON-encoded string;
	// unwrap and try again.
	var wrapper struct {
		Status json.RawMessage `json:"status"`
		Info   string          `json:"info"`
	}
	if json.Unmarshal(out, &wrapper) != nil {
		return p, false
	}
	if len(wrapper.Status) > 0 {
		_ = json.Unmarshal(wrapper.Status, &p.Status)
	}
	if wrapper.Info != "" {
		_ = json.Unmarshal([]byte(wrapper.Info), &p.Info)
	}
	return p, true
}
