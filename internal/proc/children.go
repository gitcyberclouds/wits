package proc

import (
	"fmt"
	"sort"

	"github.com/pranshuparmar/witr/pkg/model"
)

// ResolveChildren returns the direct child processes for the provided PID.
func ResolveChildren(pid int) ([]model.Process, error) {
	if pid <= 0 {
		return nil, fmt.Errorf("invalid pid")
	}

	processes, err := listProcessSnapshot()
	if err != nil {
		return nil, err
	}

	children := make([]model.Process, 0)
	for _, proc := range processes {
		if proc.PPID == pid {
			children = append(children, proc)
		}
	}

	sortProcesses(children)
	return children, nil
}

// ResolveDescendants builds a descendant tree rooted at the provided process.
func ResolveDescendants(root model.Process) (*model.ProcessTree, error) {
	if root.PID <= 0 {
		return nil, fmt.Errorf("invalid pid")
	}

	processes, err := listProcessSnapshot()
	if err != nil {
		return nil, err
	}

	childrenIndex := indexByParent(processes)
	for pid, children := range childrenIndex {
		sortProcesses(children)
		childrenIndex[pid] = children
	}

	visited := make(map[int]bool)
	var buildTree func(model.Process) model.ProcessTree
	buildTree = func(proc model.Process) model.ProcessTree {
		if visited[proc.PID] {
			return model.ProcessTree{Process: proc}
		}
		visited[proc.PID] = true

		children := childrenIndex[proc.PID]
		nodes := make([]model.ProcessTree, 0, len(children))
		for _, child := range children {
			nodes = append(nodes, buildTree(child))
		}
		return model.ProcessTree{Process: proc, Children: nodes}
	}

	rootTree := buildTree(root)
	return &rootTree, nil
}

func indexByParent(processes []model.Process) map[int][]model.Process {
	childrenIndex := make(map[int][]model.Process)
	for _, proc := range processes {
		childrenIndex[proc.PPID] = append(childrenIndex[proc.PPID], proc)
	}
	return childrenIndex
}

func sortProcesses(processes []model.Process) {
	sort.Slice(processes, func(i, j int) bool {
		return processes[i].PID < processes[j].PID
	})
}
