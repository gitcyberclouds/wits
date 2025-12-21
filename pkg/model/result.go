package model

type Result struct {
	Target   Target
	Process  Process
	Chain    []Process
	Ancestry []Process
	Source   Source
	Warnings []string
}
