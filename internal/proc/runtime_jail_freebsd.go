//go:build freebsd

package proc

import (
	"context"
	"encoding/json"
	"os/exec"
	"strconv"
	"strings"

	"github.com/pranshuparmar/witr/pkg/model"
)

func init() { registerRuntime(jailRuntime{}) }

type jailRuntime struct{}

func (jailRuntime) Name() string    { return "jail" }
func (jailRuntime) Available() bool { return binAvailable("jls") }

func (jailRuntime) List() []*model.ContainerMatch {
	if matches, ok := jailListJSON(); ok {
		return matches
	}
	return jailListText()
}

// jailListJSON uses libxo's JSON encoder (available on modern FreeBSD) so
// values containing whitespace are parsed unambiguously.
func jailListJSON() ([]*model.ContainerMatch, bool) {
	ctx, cancel := context.WithTimeout(context.Background(), runtimeQueryTimeout)
	defer cancel()
	out, err := exec.CommandContext(ctx, "jls", "--libxo=json", "jid", "name", "host.hostname", "path", "dying").Output()
	if err != nil {
		return nil, false
	}

	var payload struct {
		JailInformation struct {
			Jail []map[string]string `json:"jail"`
		} `json:"jail-information"`
	}
	if err := json.Unmarshal(out, &payload); err != nil {
		return nil, false
	}

	var matches []*model.ContainerMatch
	for _, j := range payload.JailInformation.Jail {
		state := "running"
		if j["dying"] != "" && j["dying"] != "0" {
			state = "dying"
		}
		matches = append(matches, &model.ContainerMatch{
			Runtime: "jail",
			ID:      j["jid"],
			Name:    j["name"],
			Image:   j["host.hostname"],
			Command: j["path"],
			State:   state,
			Status:  state,
		})
	}
	return matches, true
}

// jailListText is the legacy whitespace-parsing fallback for FreeBSD
// releases without libxo support in jls.
func jailListText() []*model.ContainerMatch {
	ctx, cancel := context.WithTimeout(context.Background(), runtimeQueryTimeout)
	defer cancel()
	out, err := exec.CommandContext(ctx, "jls", "-h", "jid", "name", "host.hostname", "path", "dying").Output()
	if err != nil {
		return nil
	}

	lines := strings.Split(strings.TrimSpace(string(out)), "\n")
	if len(lines) < 2 {
		return nil
	}
	var matches []*model.ContainerMatch
	for _, line := range lines[1:] {
		fields := strings.Fields(line)
		if len(fields) < 4 {
			continue
		}
		state := "running"
		if len(fields) >= 5 && fields[4] != "0" {
			state = "dying"
		}
		matches = append(matches, &model.ContainerMatch{
			Runtime: "jail",
			ID:      fields[0],
			Name:    fields[1],
			Image:   fields[2],
			Command: fields[3],
			State:   state,
			Status:  state,
		})
	}
	return matches
}

func (jailRuntime) HostPID(id string) int {
	ctx, cancel := context.WithTimeout(context.Background(), runtimeQueryTimeout)
	defer cancel()
	out, err := exec.CommandContext(ctx, "ps", "-J", id, "-o", "pid=").Output()
	if err != nil {
		return 0
	}
	for _, line := range strings.Split(strings.TrimSpace(string(out)), "\n") {
		if pid, err := strconv.Atoi(strings.TrimSpace(line)); err == nil && pid > 0 {
			return pid
		}
	}
	return 0
}
