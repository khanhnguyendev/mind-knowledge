// Package sync reconciles registered projects against the filesystem. It
// reports drift and changes nothing.
package sync

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/khanhnguyendev/mind-knowledge/internal/model"
	"github.com/khanhnguyendev/mind-knowledge/internal/store"
)

// The states a project can be in after a check.
const (
	StateOK            = "ok"
	StateMissing       = "missing"
	StateNotGit        = "not-git"
	StateRemoteChanged = "remote-changed"
)

// Result is one project's reconciliation outcome.
type Result struct {
	Project model.Project `json:"project"`
	State   string        `json:"state"`
	Branch  string        `json:"branch,omitempty"`
	Head    string        `json:"head,omitempty"`
	Detail  string        `json:"detail,omitempty"`
}

// Run checks each project in turn.
func Run(s *store.Store, projects []model.Project) []Result {
	results := make([]Result, 0, len(projects))
	for _, p := range projects {
		results = append(results, Inspect(p))
	}
	return results
}

// Inspect checks one project: does its path still exist, is it still a git
// repository, and does its remote still match what was recorded.
func Inspect(p model.Project) Result {
	res := Result{Project: p}

	info, err := os.Stat(p.RepoPath)
	if err != nil || !info.IsDir() {
		res.State = StateMissing
		res.Detail = fmt.Sprintf("%s does not exist", p.RepoPath)
		return res
	}

	if _, err := os.Stat(filepath.Join(p.RepoPath, ".git")); err != nil {
		res.State = StateNotGit
		res.Detail = fmt.Sprintf("%s is not a git repository", p.RepoPath)
		return res
	}

	res.Branch = gitOutput(p.RepoPath, "rev-parse", "--abbrev-ref", "HEAD")
	res.Head = gitOutput(p.RepoPath, "rev-parse", "HEAD")

	if p.GitRemote != "" {
		actual := gitOutput(p.RepoPath, "remote", "get-url", "origin")
		if actual != "" && actual != p.GitRemote {
			res.State = StateRemoteChanged
			res.Detail = fmt.Sprintf("recorded %s, found %s", p.GitRemote, actual)
			return res
		}
	}

	res.State = StateOK
	return res
}

// gitOutput runs a git command in dir and returns its trimmed output, or
// an empty string if the command fails.
func gitOutput(dir string, args ...string) string {
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	out, err := cmd.Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}
