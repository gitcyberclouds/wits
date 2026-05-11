//go:build darwin

package source

import (
	"strings"

	"github.com/pranshuparmar/witr/internal/launchd"
	"github.com/pranshuparmar/witr/pkg/model"
)

func detectLaunchd(ancestry []model.Process) *model.Source {
	// Check if the ancestry includes launchd (PID 1)
	hasLaunchd := false
	for _, p := range ancestry {
		if p.PID == 1 && p.Command == "launchd" {
			hasLaunchd = true
			break
		}
	}

	if !hasLaunchd {
		return nil
	}

	// Get the target process (last in ancestry)
	if len(ancestry) == 0 {
		return nil
	}
	target := ancestry[len(ancestry)-1]

	// Try to get detailed launchd info for the target process
	info, err := launchd.GetLaunchdInfo(target.PID)
	if err != nil {
		// Fall back to basic launchd detection
		return &model.Source{
			Type: model.SourceLaunchd,
			Name: "launchd",
		}
	}

	// Build the source with details
	source := &model.Source{
		Type:        model.SourceLaunchd,
		Name:        info.Label,
		Description: info.Comment,
		Details:     make(map[string]string),
	}

	// Add domain description (Launch Agent vs Launch Daemon)
	source.Details["type"] = info.DomainDescription()

	// Add plist path if found
	if info.PlistPath != "" {
		source.UnitFile = info.PlistPath
		source.Details["plist"] = info.PlistPath
	}

	// Separate schedule triggers from non-schedule triggers
	var scheduleParts, triggerParts []string
	for _, t := range info.FormatTriggers() {
		if strings.HasPrefix(t, "StartInterval") || strings.HasPrefix(t, "StartCalendarInterval") {
			scheduleParts = append(scheduleParts, t)
		} else {
			triggerParts = append(triggerParts, t)
		}
	}
	if len(scheduleParts) > 0 {
		source.Details["schedule"] = strings.Join(scheduleParts, "; ")
	}
	if len(triggerParts) > 0 {
		source.Details["triggers"] = strings.Join(triggerParts, "; ")
	}

	// Add KeepAlive status
	if info.KeepAlive {
		source.Details["keepalive"] = "Yes (restarts if killed)"
	}

	return source
}
