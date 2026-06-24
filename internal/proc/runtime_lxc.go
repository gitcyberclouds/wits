package proc

import (
	"context"
	"encoding/json"
	"os/exec"
	"strconv"
	"strings"

	"github.com/pranshuparmar/witr/pkg/model"
)

func init() { registerRuntime(lxcRuntime{}) }

type lxcRuntime struct{}

func (lxcRuntime) Name() string    { return "lxc" }
func (lxcRuntime) Available() bool { return binAvailable("lxc-ls") }

// List uses `lxc-ls --fancy --format json`, which returns a flat per-container
// record with name/state/ipv4/ipv6/groups/autostart/unprivileged. Image and
// PID aren't part of the listing — PID is filled in lazily via HostPID, image
// is left empty since classic LXC doesn't track image metadata after create.
func (lxcRuntime) List() []*model.ContainerMatch {
	ctx, cancel := context.WithTimeout(context.Background(), runtimeQueryTimeout)
	defer cancel()
	out, err := exec.CommandContext(ctx, "lxc-ls", "--fancy", "--format", "json").Output()
	if err != nil {
		return nil
	}
	return parseLXCList(out)
}

func (lxcRuntime) HostPID(id string) int {
	ctx, cancel := context.WithTimeout(context.Background(), runtimeQueryTimeout)
	defer cancel()
	out, err := exec.CommandContext(ctx, "lxc-info", "-n", id, "-p", "-H").Output()
	if err != nil {
		return 0
	}
	pid, _ := strconv.Atoi(strings.TrimSpace(string(out)))
	return pid
}

type lxcLsEntry struct {
	Name         string `json:"name"`
	State        string `json:"state"`
	Autostart    string `json:"autostart"`
	Groups       string `json:"groups"`
	IPv4         string `json:"ipv4"`
	IPv6         string `json:"ipv6"`
	Unprivileged string `json:"unprivileged"`
}

func parseLXCList(out []byte) []*model.ContainerMatch {
	var entries []lxcLsEntry
	if err := json.Unmarshal(out, &entries); err != nil {
		return nil
	}
	var matches []*model.ContainerMatch
	for _, e := range entries {
		state := strings.ToLower(e.State)
		networks := ""
		switch {
		case e.IPv4 != "" && e.IPv6 != "":
			networks = e.IPv4 + ", " + e.IPv6
		case e.IPv4 != "":
			networks = e.IPv4
		case e.IPv6 != "":
			networks = e.IPv6
		}
		matches = append(matches, &model.ContainerMatch{
			Runtime:  "lxc",
			ID:       e.Name,
			Name:     e.Name,
			State:    state,
			Status:   e.State,
			Networks: networks,
		})
	}
	return matches
}
