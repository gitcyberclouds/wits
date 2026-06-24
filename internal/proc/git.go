package proc

import (
	"os"
	"path/filepath"
	"strings"
)

func detectGitInfo(cwd string) (string, string) {
	if cwd == "" || cwd == "unknown" {
		return "", ""
	}

	searchDir := cwd
	for depth := 0; depth < 10; depth++ {
		gitPath := filepath.Join(searchDir, ".git")
		if fi, err := os.Stat(gitPath); err == nil {
			gitDir := gitPath
			if !fi.IsDir() {
				// In a worktree or submodule, .git is a file holding a
				// "gitdir: <path>" pointer to the real git directory.
				gitDir = gitDirFromFile(gitPath, searchDir)
			}
			if gitDir != "" {
				return filepath.Base(searchDir), gitBranchFromHEAD(gitDir)
			}
		}

		parent := filepath.Dir(searchDir)
		if parent == searchDir {
			break
		}
		searchDir = parent
	}

	return "", ""
}

// gitDirFromFile parses the "gitdir: <path>" pointer in a .git file (used by
// worktrees and submodules) and returns the real git directory as an absolute
// path, or "" if it can't be read.
func gitDirFromFile(gitFile, baseDir string) string {
	data, err := os.ReadFile(gitFile)
	if err != nil {
		return ""
	}
	for _, line := range strings.Split(string(data), "\n") {
		rest, ok := strings.CutPrefix(strings.TrimSpace(line), "gitdir:")
		if !ok {
			continue
		}
		dir := strings.TrimSpace(rest)
		if dir == "" {
			return ""
		}
		if !filepath.IsAbs(dir) {
			dir = filepath.Join(baseDir, dir)
		}
		return dir
	}
	return ""
}

// gitBranchFromHEAD reads <gitDir>/HEAD and returns the checked-out branch name,
// or "" when HEAD is detached or unreadable.
func gitBranchFromHEAD(gitDir string) string {
	head, err := os.ReadFile(filepath.Join(gitDir, "HEAD"))
	if err != nil {
		return ""
	}
	if ref, ok := strings.CutPrefix(strings.TrimSpace(string(head)), "ref: "); ok {
		return strings.TrimPrefix(ref, "refs/heads/")
	}
	return ""
}
