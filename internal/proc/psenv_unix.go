//go:build darwin || freebsd

package proc

import (
	"os"
	"os/exec"
	"strconv"
	"strings"
)

func buildEnvForPS() []string {
	var env []string
	for _, e := range os.Environ() {
		if !strings.HasPrefix(e, "LC_ALL=") && !strings.HasPrefix(e, "TZ=") {
			env = append(env, e)
		}
	}
	env = append(env, "LC_ALL=C", "TZ=UTC")
	return env
}

// readPIDCommMap returns a pid->comm map produced by a single `ps -axo pid=,comm=`
// invocation. On macOS comm holds the full executable path (spaces included);
// on FreeBSD it holds the short command name. Either way the value is the
// authoritative source for the display name and is parsed by splitting on the
// first whitespace only, so values containing spaces survive intact.
func readPIDCommMap() map[int]string {
	cmd := exec.Command("ps", "-axo", "pid=,comm=")
	cmd.Env = buildEnvForPS()
	out, err := cmd.Output()
	if err != nil {
		return nil
	}
	m := make(map[int]string)
	for _, line := range strings.Split(string(out), "\n") {
		trimmed := strings.TrimLeft(line, " \t")
		if trimmed == "" {
			continue
		}
		idx := strings.IndexAny(trimmed, " \t")
		if idx < 0 {
			continue
		}
		pid, err := strconv.Atoi(trimmed[:idx])
		if err != nil {
			continue
		}
		comm := strings.TrimSpace(trimmed[idx+1:])
		if comm != "" {
			m[pid] = comm
		}
	}
	return m
}
