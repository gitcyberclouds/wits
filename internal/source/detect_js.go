//go:build js

// detect_js.go supplies the init-system detectors for the WASM build.
//
// witr's container / SSH / shell / supervisor / cron / init detectors are pure
// ancestry analysis and run unchanged in the browser. The systemd, launchd, and
// bsd-rc detectors, however, read unit metadata that only exists on the host
// (systemctl, plist files, rc.d). That metadata is authored per world and
// injected here via WorldSources, keyed by the target PID.
package source

import "github.com/pranshuparmar/witr/pkg/model"

// WorldSources holds authored init-system source metadata for the current
// world, keyed by the target process PID. Set by the WASM entry point.
var WorldSources = map[int]model.Source{}

func injectedSource(ancestry []model.Process, t model.SourceType) *model.Source {
	if len(ancestry) == 0 {
		return nil
	}
	last := ancestry[len(ancestry)-1]
	if s, ok := WorldSources[last.PID]; ok && s.Type == t {
		src := s
		return &src
	}
	return nil
}

func detectSystemd(ancestry []model.Process) *model.Source {
	return injectedSource(ancestry, model.SourceSystemd)
}

func detectLaunchd(ancestry []model.Process) *model.Source {
	return injectedSource(ancestry, model.SourceLaunchd)
}

func detectBsdRc(ancestry []model.Process) *model.Source {
	return injectedSource(ancestry, model.SourceBsdRc)
}
