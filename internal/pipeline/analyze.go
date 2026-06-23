package pipeline

import (
	"sort"
	"strconv"
	"strings"

	procpkg "github.com/pranshuparmar/witr/internal/proc"
	"github.com/pranshuparmar/witr/internal/source"
	"github.com/pranshuparmar/witr/pkg/model"
)

type AnalyzeConfig struct {
	PID     int
	Verbose bool
	Tree    bool
	Target  model.Target
}

func AnalyzePID(cfg AnalyzeConfig) (model.Result, error) {
	ancestry, err := procpkg.ResolveAncestry(cfg.PID)
	if err != nil {
		return model.Result{}, err
	}

	src := source.Detect(ancestry)

	// ReadProcess labels lxc.payload cgroups generically as "lxc-based:" since
	// it can't see the ancestry. source.Detect knows the actual runtime via the
	// ancestor binary (incusd/lxd/lxc-start) — rewrite the per-process label so
	// the Container line matches the Source line.
	if src.Type == model.SourceContainer {
		switch src.Name {
		case "incus", "lxd", "lxc":
			for i := range ancestry {
				switch {
				case strings.HasPrefix(ancestry[i].Container, "lxc-based:"):
					ancestry[i].Container = src.Name + ":" + strings.TrimPrefix(ancestry[i].Container, "lxc-based:")
				case ancestry[i].Container == "lxc-based":
					ancestry[i].Container = src.Name
				}
			}
		}
	}

	var proc model.Process
	resolvedTarget := "unknown"
	if len(ancestry) > 0 {
		proc = ancestry[len(ancestry)-1]
		resolvedTarget = proc.Command
	}

	// Resolve the target container's healthcheck so the warning only fires when
	// the runtime confirms none is configured.
	if proc.ContainerID != "" {
		hc := procpkg.ContainerHealthcheckStatus(proc.ContainerID, proc.ContainerRuntime)
		proc.ContainerHealthcheck = hc
		if len(ancestry) > 0 {
			ancestry[len(ancestry)-1].ContainerHealthcheck = hc
		}
	}

	// Collect child PIDs once and reuse for both extended info and tree output
	var childPIDs []int
	var childProcesses []model.Process
	if (cfg.Verbose || cfg.Tree) && proc.PID > 0 {
		snapshot, err := procpkg.ListProcessSnapshot()
		if err == nil {
			for _, p := range snapshot {
				if p.PPID == proc.PID {
					childPIDs = append(childPIDs, p.PID)
					childProcesses = append(childProcesses, p)
				}
			}
			sort.Slice(childProcesses, func(i, j int) bool {
				return childProcesses[i].PID < childProcesses[j].PID
			})
			sort.Ints(childPIDs)
		}
	}

	if cfg.Verbose && len(ancestry) > 0 {
		memInfo, ioStats, fileDescs, fdCount, fdLimit, threadCount, err := procpkg.ReadExtendedInfo(cfg.PID)
		if err == nil {
			proc.Memory = memInfo
			proc.IO = ioStats
			proc.FileDescs = fileDescs
			proc.FDCount = fdCount
			proc.FDLimit = fdLimit
			proc.Children = childPIDs
			proc.ThreadCount = threadCount
			ancestry[len(ancestry)-1] = proc
		}
	}

	var resCtx *model.ResourceContext
	var fileCtx *model.FileContext
	if cfg.Verbose {
		resCtx = procpkg.GetResourceContext(cfg.PID)
		fileCtx = procpkg.GetFileContext(cfg.PID)
	}

	restartCount := 0
	if src.Type == model.SourceSystemd {
		if v, ok := src.Details["NRestarts"]; ok {
			if count, err := strconv.Atoi(v); err == nil {
				restartCount = count
			}
		}
	}

	res := model.Result{
		Target:          cfg.Target,
		ResolvedTarget:  resolvedTarget,
		Process:         proc,
		RestartCount:    restartCount,
		Ancestry:        ancestry,
		Source:          src,
		Warnings:        source.Warnings(ancestry, restartCount, src.Type),
		ResourceContext: resCtx,
		FileContext:     fileCtx,
		Children:        childProcesses,
	}

	return res, nil
}
