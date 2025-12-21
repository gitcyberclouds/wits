package source

import "github.com/pranshuparmar/witr/pkg/model"

var knownSupervisors = map[string]bool{
	"pm2":         true,
	"supervisord": true,
	"gunicorn":    true,
	"uwsgi":       true,
}

func detectSupervisor(ancestry []model.Process) *model.Source {
	for _, p := range ancestry {
		if knownSupervisors[p.Command] {
			return &model.Source{
				Type:       model.SourceSupervisor,
				Name:       p.Command,
				Confidence: 0.7,
			}
		}
	}
	return nil
}
