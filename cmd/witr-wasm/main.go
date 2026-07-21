//go:build js && wasm

// Command witr-wasm exposes witr's real engine to the browser.
//
// It loads an in-memory "world" (the same worlds/*.json the playground uses),
// then runs targets through witr's actual resolve → analyze → render pipeline —
// the identical Go code the CLI runs, compiled to WebAssembly. Only the data
// source (internal/proc/world_js.go) and the init-system metadata
// (internal/source/detect_js.go) are browser-specific; everything above is witr.
//
// JS calls the global function installed here:
//
//	witrRun(worldJSON: string, nowMs: number, argv: string[])
//	  -> { output: string, exit: number }
package main

import (
	"bytes"
	"encoding/json"
	"sort"
	"strconv"
	"strings"
	"syscall/js"
	"time"

	"github.com/pranshuparmar/witr/internal/output"
	"github.com/pranshuparmar/witr/internal/pipeline"
	procpkg "github.com/pranshuparmar/witr/internal/proc"
	"github.com/pranshuparmar/witr/internal/source"
	"github.com/pranshuparmar/witr/internal/target"
	"github.com/pranshuparmar/witr/pkg/model"
)

func init() {
	// Force UTC so time formatting never reads /etc/localtime (there is no
	// filesystem in the browser) and matches the playground's UTC rendering.
	time.Local = time.UTC
}

func main() {
	js.Global().Set("witrRun", js.FuncOf(witrRun))
	js.Global().Set("witrReady", js.ValueOf(true))
	select {} // keep the Go runtime alive for callbacks
}

func witrRun(_ js.Value, args []js.Value) any {
	worldJSON := args[0].String()
	nowMs := int64(args[1].Float())
	argv := make([]string, args[2].Length())
	for i := range argv {
		argv[i] = args[2].Index(i).String()
	}

	now := time.UnixMilli(nowMs).UTC()
	if err := loadWorld(worldJSON, now); err != nil {
		return result("error: "+err.Error()+"\n", 5)
	}

	text, exit := run(argv)
	return result(text, exit)
}

func result(text string, exit int) any {
	return map[string]any{"output": text, "exit": exit}
}

// ---- world loading ----------------------------------------------------------

type wSocket struct {
	Address, Protocol, State string
	Port                     int
}
type wSource struct {
	Type, Name, Description, UnitFile string
	Details                           map[string]string
}
type wProc struct {
	PID, PPID                      int
	Command, Cmdline, User         string
	StartedAgo                     int64
	WorkingDir, GitRepo, GitBranch string
	Container, Service             string
	Forked, Health                 string
	Sockets                        []wSocket
	Env, Capabilities              []string
	LockedFiles                    []string
	ThreadCount, FDCount, FDLimit  int
	Memory                         *struct{ VMS, RSS, Shared uint64 }
	IO                             *struct{ ReadBytes, WriteBytes, ReadOps, WriteOps uint64 }
	Source                         *wSource
}
type wLock struct {
	PID           int
	Process, Path string
	Type, Mode    string
}
type wContainer struct {
	Runtime, ID, Name, Image, Command    string
	State, Status, Health                string
	CreatedAgo, StartedAgo               int64
	Networks, Mounts, Ports              string
	ComposeProject, ComposeService       string
	ComposeConfigFile, ComposeWorkingDir string
}
type wWorld struct {
	Processes       []wProc      `json:"processes"`
	Locks           []wLock      `json:"locks"`
	Containers      []wContainer `json:"containers"`
	SocketOverrides map[string]struct {
		State, Explanation, Workaround string
	} `json:"socketOverrides"`
}

// worldContainers holds the current world's containers for the -c path.
var worldContainers []*model.ContainerMatch

func loadWorld(raw string, now time.Time) error {
	var w wWorld
	if err := json.Unmarshal([]byte(raw), &w); err != nil {
		return err
	}

	procs := make([]model.Process, 0, len(w.Processes))
	sources := map[int]model.Source{}
	var ports []model.OpenPort

	for _, p := range w.Processes {
		mp := model.Process{
			PID: p.PID, PPID: p.PPID, Command: p.Command, Cmdline: p.Cmdline,
			User: p.User, WorkingDir: p.WorkingDir, GitRepo: p.GitRepo, GitBranch: p.GitBranch,
			Container: p.Container, Service: p.Service, Forked: p.Forked, Health: p.Health,
			Env: p.Env, Capabilities: p.Capabilities,
			ThreadCount: p.ThreadCount, FDCount: p.FDCount, FDLimit: uint64(p.FDLimit),
		}
		if p.StartedAgo != 0 {
			mp.StartedAt = now.Add(-time.Duration(p.StartedAgo) * time.Second)
		}
		for _, s := range p.Sockets {
			mp.Sockets = append(mp.Sockets, model.Socket{Address: s.Address, Port: s.Port, Protocol: s.Protocol, State: s.State})
			if s.State == "LISTEN" {
				ports = append(ports, model.OpenPort{PID: p.PID, Port: s.Port, Address: s.Address, Protocol: s.Protocol, State: s.State})
			}
		}
		if p.Memory != nil {
			mp.Memory = model.MemoryInfo{VMS: p.Memory.VMS, RSS: p.Memory.RSS, Shared: p.Memory.Shared}
		}
		if p.IO != nil {
			mp.IO = model.IOStats{ReadBytes: p.IO.ReadBytes, WriteBytes: p.IO.WriteBytes, ReadOps: p.IO.ReadOps, WriteOps: p.IO.WriteOps}
		}
		procs = append(procs, mp)

		// Only systemd/launchd/bsd-rc metadata is injected; the other source
		// types are detected for real from the ancestry.
		if p.Source != nil {
			st := model.SourceType(p.Source.Type)
			switch st {
			case model.SourceSystemd, model.SourceLaunchd, model.SourceBsdRc:
				sources[p.PID] = model.Source{Type: st, Name: p.Source.Name, Description: p.Source.Description, UnitFile: p.Source.UnitFile, Details: p.Source.Details}
			}
		}
	}

	var locks []*model.LockedFile
	for _, l := range w.Locks {
		locks = append(locks, &model.LockedFile{PID: l.PID, Process: l.Process, Path: l.Path, Type: l.Type, Mode: l.Mode})
	}

	sockets := map[int]*model.SocketInfo{}
	for k, v := range w.SocketOverrides {
		if port, err := strconv.Atoi(k); err == nil {
			sockets[port] = &model.SocketInfo{Port: port, State: v.State, Explanation: v.Explanation, Workaround: v.Workaround}
		}
	}

	worldContainers = nil
	for _, c := range w.Containers {
		m := &model.ContainerMatch{
			Runtime: c.Runtime, ID: c.ID, Name: c.Name, Image: c.Image, Command: c.Command,
			State: c.State, Status: c.Status, Health: c.Health,
			Networks: c.Networks, Mounts: c.Mounts, Ports: c.Ports,
			ComposeProject: c.ComposeProject, ComposeService: c.ComposeService,
			ComposeConfigFile: c.ComposeConfigFile, ComposeWorkingDir: c.ComposeWorkingDir,
		}
		if c.StartedAgo != 0 {
			m.StartedAt = now.Add(-time.Duration(c.StartedAgo) * time.Second)
		}
		if c.CreatedAgo != 0 {
			m.CreatedAt = now.Add(-time.Duration(c.CreatedAgo) * time.Second)
		}
		worldContainers = append(worldContainers, m)
	}

	procpkg.LoadWorld(procs, locks, ports, sockets)
	source.WorldSources = sources
	return nil
}

func resolveContainer(query string, exact bool) []*model.ContainerMatch {
	lower := strings.ToLower(query)
	var out []*model.ContainerMatch
	for _, c := range worldContainers {
		if exact {
			if strings.EqualFold(c.Name, query) {
				out = append(out, c)
			}
			continue
		}
		for _, f := range []string{c.Name, c.Image, c.Command, c.ComposeProject, c.ComposeService} {
			if strings.Contains(strings.ToLower(f), lower) {
				out = append(out, c)
				break
			}
		}
	}
	return out
}

func containerByPort(port int) *model.ContainerMatch {
	for _, c := range worldContainers {
		for _, seg := range strings.Split(c.Ports, ",") {
			if idx := strings.Index(seg, "->"); idx > 0 {
				hostside := seg[:idx]
				if colon := strings.LastIndexByte(hostside, ':'); colon >= 0 {
					if n, err := strconv.Atoi(strings.TrimSpace(hostside[colon+1:])); err == nil && n == port {
						return c
					}
				}
			}
		}
	}
	return nil
}

// ---- routing (mirrors internal/app/app.go for the paths the playground uses) --

type flags struct {
	short, tree, json, env, warnings, verbose, exact, color bool
}

type tgt struct {
	kind, value string
}

func run(argv []string) (string, int) {
	targets, fl := parseArgs(argv)
	multi := len(targets) > 1
	var out bytes.Buffer
	highest := 0

	for i, t := range targets {
		if multi {
			if i > 0 {
				out.WriteByte('\n')
			}
			writeDivider(&out, t, fl.color)
		}
		exit := oneTarget(&out, t, fl, multi)
		if exit > highest {
			highest = exit
		}
	}
	if len(targets) == 0 {
		return "usage: witr [flags] [name...]\n", 4
	}
	return out.String(), highest
}

func oneTarget(out *bytes.Buffer, t tgt, fl flags, multi bool) int {
	if t.kind == "container" {
		return containerTarget(out, t, fl)
	}

	var pids []int
	var err error
	switch t.kind {
	case "pid":
		n, _ := strconv.Atoi(t.value)
		if _, e := procpkg.ReadProcess(n); e == nil {
			pids = []int{n}
		} else {
			err = e
		}
	case "port":
		n, _ := strconv.Atoi(t.value)
		pids, err = target.ResolvePort(n)
		if len(pids) == 0 {
			// A published port with no host process falls back to the container.
			if c := containerByPort(n); c != nil {
				return renderContainer(out, "port "+t.value, c, fl)
			}
		}
	case "file":
		pids, err = target.ResolveFile(t.value)
	default:
		pids, err = target.ResolveName(t.value, fl.exact)
	}

	if err != nil || len(pids) == 0 {
		out.WriteString("Error: no matching process or service found. Please check your query or try a different name/port/PID.\n")
		return 2
	}
	if len(pids) > 1 {
		writeMultiMatch(out, pids, fl.color)
		return 4
	}

	res, aerr := pipeline.AnalyzePID(pipeline.AnalyzeConfig{
		PID: pids[0], Verbose: fl.verbose, Tree: fl.tree,
		Target: model.Target{Type: targetType(t.kind), Value: t.value},
	})
	if aerr != nil {
		out.WriteString("error: " + aerr.Error() + "\n")
		return 5
	}
	if t.kind == "port" && fl.verbose {
		res.SocketInfo = procpkg.GetSocketStateForPort(atoi(t.value))
		source.EnrichSocketInfo(res.SocketInfo)
	}

	switch {
	case fl.json:
		s, _ := output.ToJSON(res)
		out.WriteString(s + "\n")
	case fl.env:
		output.RenderEnvOnly(out, res, fl.color)
	case fl.short:
		output.RenderShort(out, res, fl.color)
	case fl.tree:
		output.PrintTree(out, res.Ancestry, res.Children, fl.color)
	case fl.warnings:
		output.RenderWarnings(out, res, fl.color)
	default:
		output.RenderStandard(out, res, fl.color, fl.verbose)
	}
	if len(res.Warnings) > 0 {
		return 1
	}
	return 0
}

func containerTarget(out *bytes.Buffer, t tgt, fl flags) int {
	matches := resolveContainer(t.value, fl.exact)
	if len(matches) == 0 {
		out.WriteString("Error: no matching container found.\n")
		return 2
	}
	if len(matches) > 1 {
		out.WriteString("Multiple matching containers found:\n\n")
		for i, m := range matches {
			if fl.color {
				out.WriteString("[" + strconv.Itoa(i+1) + "] " + string(output.ColorGreen) + m.Name + string(output.ColorReset) + " (" + string(output.ColorDim) + m.Runtime + string(output.ColorReset) + ")\n")
			} else {
				out.WriteString("[" + strconv.Itoa(i+1) + "] " + m.Name + " (" + m.Runtime + ")\n")
			}
			detail := "image: " + m.Image
			if m.Status != "" {
				detail += ", status: " + m.Status
			}
			if m.Ports != "" {
				detail += ", ports: " + m.Ports
			}
			out.WriteString("    " + detail + "\n")
		}
		out.WriteString("\nRe-run with the exact container name to disambiguate:\n  witr -c <container-name> --exact\n")
		return 4
	}
	return renderContainer(out, "container "+t.value, matches[0], fl)
}

func renderContainer(out *bytes.Buffer, label string, m *model.ContainerMatch, fl flags) int {
	switch {
	case fl.json:
		s, _ := output.ContainerFallbackToJSON(label, m)
		out.WriteString(s + "\n")
	case fl.short:
		output.RenderContainerFallbackShort(out, label, m, fl.color)
	case fl.tree:
		output.RenderContainerFallbackTree(out, m, fl.color)
	case fl.warnings:
		output.RenderContainerFallbackWarnings(out, m, fl.color)
	default:
		output.RenderContainerFallback(out, label, m, fl.color, fl.verbose)
	}
	return 0
}

func writeDivider(out *bytes.Buffer, t tgt, color bool) {
	label := t.kind + ": " + t.value
	if t.kind == "name" {
		label = "name: " + t.value
	}
	if color {
		out.WriteString(string(output.ColorCyan) + "----- [" + label + "] -----" + string(output.ColorReset) + "\n")
	} else {
		out.WriteString("----- [" + label + "] -----\n")
	}
}

func writeMultiMatch(out *bytes.Buffer, pids []int, color bool) {
	out.WriteString("Multiple matching processes found:\n\n")
	for i, pid := range pids {
		p, _ := procpkg.ReadProcess(pid)
		if color {
			out.WriteString("[" + strconv.Itoa(i+1) + "] " + string(output.ColorGreen) + p.Command + string(output.ColorReset) + " (" + string(output.ColorDim) + "pid " + strconv.Itoa(pid) + string(output.ColorReset) + ")\n    " + p.Cmdline + "\n")
		} else {
			out.WriteString("[" + strconv.Itoa(i+1) + "] " + p.Command + " (pid " + strconv.Itoa(pid) + ")\n    " + p.Cmdline + "\n")
		}
	}
	out.WriteString("\nRe-run with:\n  witr --pid <pid>\n")
}

func targetType(kind string) model.TargetType {
	switch kind {
	case "pid":
		return model.TargetPID
	case "port":
		return model.TargetPort
	case "file":
		return model.TargetFile
	case "container":
		return model.TargetContainer
	default:
		return model.TargetName
	}
}

func atoi(s string) int { n, _ := strconv.Atoi(s); return n }

// ---- arg parsing (mirrors playground/js/parser.js) --------------------------

func parseArgs(argv []string) ([]tgt, flags) {
	fl := flags{color: true}
	var targets []tgt
	valueFlags := map[string]string{"--pid": "pid", "-p": "pid", "--port": "port", "-o": "port", "--file": "file", "-f": "file", "--container": "container", "-c": "container"}
	boolFlags := map[string]*bool{
		"--short": &fl.short, "-s": &fl.short, "--tree": &fl.tree, "-t": &fl.tree,
		"--json": &fl.json, "--env": &fl.env, "--warnings": &fl.warnings, "--verbose": &fl.verbose,
		"--exact": &fl.exact, "-x": &fl.exact,
	}
	noColor := false
	push := func(kind, raw string) {
		for _, v := range strings.Split(raw, ",") {
			if v = strings.TrimSpace(v); v != "" {
				targets = append(targets, tgt{kind, v})
			}
		}
	}
	for i := 0; i < len(argv); i++ {
		a := argv[i]
		if a == "--no-color" {
			noColor = true
			continue
		}
		if strings.HasPrefix(a, "--") && strings.Contains(a, "=") {
			eq := strings.IndexByte(a, '=')
			name, val := a[:eq], a[eq+1:]
			if k, ok := valueFlags[name]; ok {
				push(k, val)
			} else if b, ok := boolFlags[name]; ok {
				*b = true
			}
			continue
		}
		if b, ok := boolFlags[a]; ok {
			*b = true
			continue
		}
		if k, ok := valueFlags[a]; ok {
			if i+1 < len(argv) {
				push(k, argv[i+1])
				i++
			}
			continue
		}
		if len(a) > 2 && a[0] == '-' && a[1] != '-' {
			if k, ok := valueFlags[a[:2]]; ok {
				push(k, strings.TrimPrefix(a[2:], "="))
				continue
			}
		}
		if strings.HasPrefix(a, "-") {
			continue // unknown flag
		}
		push("name", a)
	}
	fl.color = !noColor
	sort.SliceStable(targets, func(i, j int) bool { return false })
	return targets, fl
}
