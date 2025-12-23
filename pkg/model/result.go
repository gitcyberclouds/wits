package model

type Result struct {
	Target         Target
	ResolvedTarget string
	Process        Process
	Ancestry       []Process
	Source         Source
	Warnings       []string
	RestartCount   int // Number of restarts detected in ancestry
}
