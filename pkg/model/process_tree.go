package model

// ProcessTree represents a minimal process hierarchy for child/descendant output.
type ProcessTree struct {
	Process  Process
	Children []ProcessTree
}
