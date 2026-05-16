package proc

import (
	"strings"

	"github.com/pranshuparmar/witr/pkg/model"
)

// ContainerRuntime is a backend (docker, podman, …) that can list containers
// and resolve a container's main host PID.
type ContainerRuntime interface {
	Name() string
	Available() bool
	List() []*model.ContainerMatch
	HostPID(id string) int
}

var registeredRuntimes []ContainerRuntime

func registerRuntime(rt ContainerRuntime) {
	registeredRuntimes = append(registeredRuntimes, rt)
}

// ResolveContainer queries every available container runtime and returns the
// merged set of matches against the query. Match predicate is substring
// (case-insensitive) across name, image, and command, unless exact is true in
// which case any of those fields must equal the query.
func ResolveContainer(query string, exact bool) []*model.ContainerMatch {
	q := strings.ToLower(query)
	var out []*model.ContainerMatch
	seen := make(map[string]bool)
	for _, rt := range registeredRuntimes {
		if !rt.Available() {
			continue
		}
		for _, c := range rt.List() {
			if !matchContainer(c, q, exact) {
				continue
			}
			key := rt.Name() + "|" + c.ID
			if seen[key] {
				continue
			}
			seen[key] = true
			out = append(out, c)
		}
	}
	return out
}

func matchContainer(c *model.ContainerMatch, query string, exact bool) bool {
	fields := []string{
		strings.ToLower(c.Name),
		strings.ToLower(c.Image),
		strings.ToLower(c.Command),
		strings.ToLower(c.ComposeProject),
		strings.ToLower(c.ComposeService),
	}
	for _, f := range fields {
		if f == "" {
			continue
		}
		if exact {
			if f == query {
				return true
			}
		} else if strings.Contains(f, query) {
			return true
		}
	}
	return false
}

// ListAllContainers returns every container reported by every available
// runtime, deduped by runtime|id. Used by the TUI's Containers tab.
func ListAllContainers() []*model.ContainerMatch {
	var out []*model.ContainerMatch
	seen := make(map[string]bool)
	for _, rt := range registeredRuntimes {
		if !rt.Available() {
			continue
		}
		for _, c := range rt.List() {
			key := rt.Name() + "|" + c.ID
			if seen[key] {
				continue
			}
			seen[key] = true
			out = append(out, c)
		}
	}
	return out
}

// ResolveContainerHostPID returns the PID of the container's main process on
// the host. Returns 0 if the runtime can't be reached or the PID isn't
// available (container not running, namespaced PID, etc.).
func ResolveContainerHostPID(runtime, id string) int {
	for _, rt := range registeredRuntimes {
		if rt.Name() == runtime && rt.Available() {
			return rt.HostPID(id)
		}
	}
	return 0
}

// enrichingRuntime is implemented by runtimes that can supply additional
// per-container details via a follow-up call, used when a single match is
// resolved and the caller wants richer metadata than the initial list scan
// produced.
type enrichingRuntime interface {
	Enrich(*model.ContainerMatch)
}

// EnrichContainer asks the originating runtime to fill in any extra fields
// available via a per-container query (e.g. crictl inspect). No-op when the
// runtime doesn't expose extra detail.
func EnrichContainer(match *model.ContainerMatch) {
	if match == nil {
		return
	}
	for _, rt := range registeredRuntimes {
		if rt.Name() != match.Runtime || !rt.Available() {
			continue
		}
		if e, ok := rt.(enrichingRuntime); ok {
			e.Enrich(match)
		}
		return
	}
}
