package model

type Result struct {
	Target         Target
	ResolvedTarget string
	Process        Process
	RestartCount   int
	Ancestry       []Process
	Source         Source
	Warnings       []string
}
