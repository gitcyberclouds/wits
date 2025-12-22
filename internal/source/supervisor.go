package source

import (
	"strings"

	"github.com/pranshuparmar/witr/pkg/model"
)

var knownSupervisors = map[string]bool{
	"pm2":         true,
	"supervisord": true,
	"gunicorn":    true,
	"uwsgi":       true,
	"pm2 god":     true,
}

func detectSupervisor(ancestry []model.Process) *model.Source {
	for _, p := range ancestry {
		if knownSupervisors[strings.ToLower(p.Command)] {
			return &model.Source{
				Type:       model.SourceSupervisor,
				Name:       p.Command,
				Confidence: 0.7,
			}
		}
		// Also match on command line for supervisor keywords
		for sup := range knownSupervisors {
			if strings.Contains(strings.ToLower(p.Cmdline), sup) {
				return &model.Source{
					Type:       model.SourceSupervisor,
					Name:       sup,
					Confidence: 0.7,
				}
			}
		}
	}
	return nil
}
