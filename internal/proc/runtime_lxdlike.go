package proc

import (
	"context"
	"encoding/json"
	"os/exec"
	"strings"
	"time"

	"github.com/pranshuparmar/witr/pkg/model"
)

// Incus and LXD share the same REST API shape (Incus is a fork of LXD), so
// `incus list --format json` and `lxc list --format json` return the same
// payload. Shared helpers here parameterize on the binary so each runtime
// stays a thin registration.

type lxdLikeInstance struct {
	Name            string                       `json:"name"`
	Type            string                       `json:"type"`
	Status          string                       `json:"status"`
	CreatedAt       string                       `json:"created_at"`
	Description     string                       `json:"description"`
	Project         string                       `json:"project"`
	Config          map[string]string            `json:"config"`
	ExpandedDevices map[string]map[string]string `json:"expanded_devices"`
	State           struct {
		Status  string                         `json:"status"`
		Pid     int                            `json:"pid"`
		Network map[string]lxdLikeNetworkEntry `json:"network"`
	} `json:"state"`
}

type lxdLikeNetworkEntry struct {
	Addresses []struct {
		Family  string `json:"family"`
		Address string `json:"address"`
		Scope   string `json:"scope"`
	} `json:"addresses"`
}

func lxdLikeList(bin, runtime string) []*model.ContainerMatch {
	ctx, cancel := context.WithTimeout(context.Background(), runtimeQueryTimeout)
	defer cancel()
	out, err := exec.CommandContext(ctx, bin, "list", "--format", "json").Output()
	if err != nil {
		return nil
	}
	return parseLXDLikeList(out, runtime)
}

func lxdLikeHostPID(bin, id string) int {
	if !isValidContainerID(id) {
		return 0
	}
	ctx, cancel := context.WithTimeout(context.Background(), runtimeQueryTimeout)
	defer cancel()
	out, err := exec.CommandContext(ctx, bin, "list", id, "--format", "json").Output()
	if err != nil {
		return 0
	}
	var instances []lxdLikeInstance
	if err := json.Unmarshal(out, &instances); err != nil {
		return 0
	}
	for _, in := range instances {
		if in.Name == id {
			return in.State.Pid
		}
	}
	if len(instances) > 0 {
		return instances[0].State.Pid
	}
	return 0
}

func lxdLikeEnrich(bin string, match *model.ContainerMatch) {
	if match == nil || match.Name == "" || !isValidContainerID(match.Name) {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), runtimeQueryTimeout)
	defer cancel()
	out, err := exec.CommandContext(ctx, bin, "list", match.Name, "--format", "json").Output()
	if err != nil {
		return
	}
	var instances []lxdLikeInstance
	if json.Unmarshal(out, &instances) != nil {
		return
	}
	for _, in := range instances {
		if in.Name != match.Name {
			continue
		}
		if nets := formatLXDLikeNetworks(in.State.Network); nets != "" {
			match.Networks = nets
		}
		if mounts := formatLXDLikeMounts(in.ExpandedDevices); mounts != "" {
			match.Mounts = mounts
		}
		return
	}
}

func parseLXDLikeList(out []byte, runtime string) []*model.ContainerMatch {
	var instances []lxdLikeInstance
	if err := json.Unmarshal(out, &instances); err != nil {
		return nil
	}
	var matches []*model.ContainerMatch
	for _, in := range instances {
		image := in.Config["image.description"]
		if image == "" {
			if os := in.Config["image.os"]; os != "" {
				image = strings.TrimSpace(os + " " + in.Config["image.release"])
			}
		}
		state := strings.ToLower(in.Status)
		created, _ := time.Parse(time.RFC3339Nano, in.CreatedAt)
		matches = append(matches, &model.ContainerMatch{
			Runtime:   runtime,
			ID:        in.Name,
			Name:      in.Name,
			Image:     image,
			State:     state,
			Status:    in.Status,
			CreatedAt: created,
		})
	}
	return matches
}

func formatLXDLikeNetworks(networks map[string]lxdLikeNetworkEntry) string {
	var parts []string
	for iface, entry := range networks {
		if iface == "lo" {
			continue
		}
		for _, addr := range entry.Addresses {
			if addr.Scope == "link" || addr.Scope == "local" {
				continue
			}
			parts = append(parts, iface+": "+addr.Address)
		}
	}
	return strings.Join(parts, ", ")
}

func formatLXDLikeMounts(devices map[string]map[string]string) string {
	var parts []string
	for name, dev := range devices {
		if dev["type"] != "disk" {
			continue
		}
		source := dev["source"]
		dest := dev["path"]
		if source == "" || dest == "" {
			continue
		}
		entry := source + " → " + dest
		if dev["readonly"] == "true" {
			entry += " (ro)"
		}
		parts = append(parts, name+": "+entry)
	}
	return strings.Join(parts, ", ")
}
