//go:build darwin || freebsd

package proc

import (
	"os"
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
