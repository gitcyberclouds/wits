//go:build freebsd

package source

import "github.com/pranshuparmar/witr/pkg/model"

func detectBsdRc(ancestry []model.Process) *model.Source {
	// Check if any process in the ancestry chain has a detected rc.d service
	for _, p := range ancestry {
		if p.Service != "" {
			return &model.Source{
				Type:       model.SourceBsdRc,
				Name:       p.Service,
				Confidence: 0.8,
				Details: map[string]string{
					"service": p.Service,
				},
			}
		}
	}

	// Check for init (PID 1) as a fallback indicator for rc.d managed processes
	// On FreeBSD, PID 1 is typically init, and rc.d services are managed by it
	for _, p := range ancestry {
		if p.PID == 1 && (p.Command == "init" || p.Command == "/sbin/init") {
			// Found init but no specific service name
			// This indicates it's likely an rc.d service but we couldn't determine which one
			return &model.Source{
				Type:       model.SourceBsdRc,
				Name:       "bsdrc",
				Confidence: 0.5,
			}
		}
	}

	return nil
}
