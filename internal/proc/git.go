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
		gitDir := filepath.Join(searchDir, ".git")
		if fi, err := os.Stat(gitDir); err == nil && fi.IsDir() {
			gitRepo := filepath.Base(searchDir)

			gitBranch := ""
			headFile := filepath.Join(gitDir, "HEAD")
			if head, err := os.ReadFile(headFile); err == nil {
				headStr := strings.TrimSpace(string(head))
				if strings.HasPrefix(headStr, "ref: ") {
					ref := strings.TrimPrefix(headStr, "ref: ")
					gitBranch = strings.TrimPrefix(ref, "refs/heads/")
				}
			}

			return gitRepo, gitBranch
		}

		parent := filepath.Dir(searchDir)
		if parent == searchDir {
			break
		}
		searchDir = parent
	}

	return "", ""
}
