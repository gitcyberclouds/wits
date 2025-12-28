//go:build darwin

package source

import "github.com/pranshuparmar/witr/pkg/model"

func detectLaunchd(ancestry []model.Process) *model.Source {
	for _, p := range ancestry {
		// On macOS, PID 1 is launchd
		if p.PID == 1 && p.Command == "launchd" {
			return &model.Source{
				Type:       model.SourceLaunchd,
				Name:       "launchd",
				Confidence: 0.8,
			}
		}
	}
	return nil
}
