//go:build freebsd

package source

import (
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/pranshuparmar/witr/pkg/model"
)

var (
	shellCache     map[string]bool
	shellCacheOnce sync.Once
)

// loadShellsFromEtc reads /etc/shells and returns a map of valid shells
func loadShellsFromEtc() map[string]bool {
	shells := make(map[string]bool)

	// Fallback list in case /etc/shells is not readable
	fallback := []string{"sh", "bash", "zsh", "csh", "tcsh", "ksh", "fish", "dash"}
	for _, s := range fallback {
		shells[s] = true
	}

	// Read /etc/shells
	data, err := os.ReadFile("/etc/shells")
	if err != nil {
		return shells
	}

	// Parse /etc/shells (one shell per line, skip comments)
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		// Extract basename (e.g., /bin/bash -> bash)
		shellName := filepath.Base(line)
		shells[shellName] = true
	}

	return shells
}

// getShells returns the cached shell list
func getShells() map[string]bool {
	shellCacheOnce.Do(func() {
		shellCache = loadShellsFromEtc()
	})
	return shellCache
}

func detectBsdRc(ancestry []model.Process) *model.Source {
	// Priority 1: Check for explicit service detection via /var/run/*.pid
	for _, p := range ancestry {
		if p.Service != "" {
			return &model.Source{
				Type:       model.SourceBsdRc,
				Name:       p.Service,
				Confidence: 0.8,
				Details: map[string]string{
					"service": p.Service,
				},
			}
		}
	}

	// Priority 2: Check if target process is a direct child of init
	// without any shell in the ancestry (likely an rc.d service)
	if len(ancestry) >= 2 {
		target := ancestry[len(ancestry)-1]
		shells := getShells()

		// Check if any ancestor (excluding target) is a shell
		hasShell := false
		for i := 0; i < len(ancestry)-1; i++ {
			if shells[ancestry[i].Command] {
				hasShell = true
				break
			}
		}

		// If target is a direct child of init (PPID=1) and no shell in ancestry
		if target.PPID == 1 && !hasShell {
			return &model.Source{
				Type:       model.SourceBsdRc,
				Name:       "bsdrc",
				Confidence: 0.6,
			}
		}
	}

	return nil
}
