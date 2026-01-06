package model

type SourceType string

const (
	SourceContainer  SourceType = "container"
	SourceSystemd    SourceType = "systemd"
	SourceLaunchd    SourceType = "launchd"
	SourceSupervisor SourceType = "supervisor"
	SourceCron       SourceType = "cron"
	SourceShell      SourceType = "shell"
	SourceUnknown    SourceType = "unknown"
)

type Source struct {
	Type    SourceType
	Name    string
	Details map[string]string
}
