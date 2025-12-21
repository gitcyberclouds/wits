package model

type Result struct {
	Target   Target
	Process  Process
	Ancestry []Process
	Source   Source
	Warnings []string
}
