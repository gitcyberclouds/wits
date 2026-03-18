package version

import (
	_ "embed"
	"runtime/debug"
	"strings"
)

//go:embed VERSION
var embedded string

// Version, Commit, and BuildDate are set via ldflags at build time.
// When ldflags are not provided (e.g., Debian packaging, go install),
// the embedded VERSION file and VCS build info are used as fallbacks.
var (
	Version   = ""
	Commit    = ""
	BuildDate = ""
)

func init() {
	if Version == "" {
		v := strings.TrimSpace(embedded)
		if v != "" {
			Version = "v" + v
		}
	}

	if Commit == "" || BuildDate == "" {
		if info, ok := debug.ReadBuildInfo(); ok {
			for _, s := range info.Settings {
				switch s.Key {
				case "vcs.revision":
					if Commit == "" && len(s.Value) >= 7 {
						Commit = s.Value[:7]
					}
				case "vcs.time":
					if BuildDate == "" {
						BuildDate = s.Value
					}
				}
			}
		}
	}

	if Version == "" {
		Version = "v0.0.0-dev"
	}
	if Commit == "" {
		Commit = "unknown"
	}
	if BuildDate == "" {
		BuildDate = "unknown"
	}
}
