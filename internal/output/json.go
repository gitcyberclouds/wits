package output

import (
	"encoding/json"

	"github.com/pranshuparmar/witr/pkg/model"
)

func ToJSON(r model.Result) (string, error) {
	data, err := json.MarshalIndent(r, "", "  ")
	if err != nil {
		return "", err
	}
	return string(data), nil
}

type shortProcess struct {
	PID     int
	Command string
}

func ToShortJSON(r model.Result) (string, error) {
	ancestry := make([]shortProcess, len(r.Ancestry))
	for i, p := range r.Ancestry {
		ancestry[i] = shortProcess{PID: p.PID, Command: p.Command}
	}
	data, err := json.MarshalIndent(ancestry, "", "  ")
	if err != nil {
		return "", err
	}
	return string(data), nil
}

func ToTreeJSON(r model.Result) (string, error) {
	type treeResult struct {
		Ancestry []shortProcess
		Children []shortProcess `json:",omitempty"`
	}

	res := treeResult{
		Ancestry: make([]shortProcess, len(r.Ancestry)),
	}

	for i, p := range r.Ancestry {
		res.Ancestry[i] = shortProcess{PID: p.PID, Command: p.Command}
	}

	if len(r.Children) > 0 {
		res.Children = make([]shortProcess, len(r.Children))
		for i, p := range r.Children {
			res.Children[i] = shortProcess{PID: p.PID, Command: p.Command}
		}
	}

	data, err := json.MarshalIndent(res, "", "  ")
	if err != nil {
		return "", err
	}
	return string(data), nil
}

func ToWarningsJSON(r model.Result) (string, error) {
	type warningResult struct {
		PID      int
		Process  string
		Command  string
		Warnings []string
	}

	procName := "unknown"
	if len(r.Ancestry) > 0 {
		procName = r.Ancestry[len(r.Ancestry)-1].Command
	} else if r.Process.Command != "" {
		procName = r.Process.Command
	}

	cmdLine := r.Process.Cmdline
	if cmdLine == "" {
		cmdLine = r.Process.Command
	}

	warnings := r.Warnings
	if warnings == nil {
		warnings = []string{}
	}

	res := warningResult{
		PID:      r.Process.PID,
		Process:  procName,
		Command:  cmdLine,
		Warnings: warnings,
	}

	data, err := json.MarshalIndent(res, "", "  ")
	if err != nil {
		return "", err
	}
	return string(data), nil
}

func ToEnvJSON(r model.Result) (string, error) {
	type envResult struct {
		PID     int
		Process string
		Command string
		Env     []string
	}

	procName := "unknown"
	if len(r.Ancestry) > 0 {
		procName = r.Ancestry[len(r.Ancestry)-1].Command
	} else if r.Process.Command != "" {
		procName = r.Process.Command
	}

	res := envResult{
		PID:     r.Process.PID,
		Process: procName,
		Command: r.Process.Cmdline,
		Env:     r.Process.Env,
	}

	data, err := json.MarshalIndent(res, "", "  ")
	if err != nil {
		return "", err
	}
	return string(data), nil
}
