// Package sync reconciles registered projects against the filesystem. It
// reports drift and changes nothing.
package sync

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/khanhnguyendev/mind-knowledge/internal/model"
)

// The states a project can be in after a check.
const (
	StateOK            = "ok"
	StateMissing       = "missing"
	StateNotGit        = "not-git"
	StateRemoteChanged = "remote-changed"
	// StateCheckFailed means the path is a git repository but git itself
	// could not be run against it — the binary is missing, or the
	// invocation timed out. This is deliberately distinct from a plain
	// "ok" with an empty branch/head, which is what a healthy repository
	// with no commits yet also looks like: a broken environment must
	// never render identically to a legitimately empty one.
	StateCheckFailed = "check-failed"
)

// gitTimeout bounds how long a single git invocation may run. It is a var
// rather than a const so tests can shorten it instead of waiting out the
// real timeout.
var gitTimeout = 5 * time.Second

// Result is one project's reconciliation outcome.
type Result struct {
	Project model.Project `json:"project"`
	State   string        `json:"state"`
	Branch  string        `json:"branch,omitempty"`
	Head    string        `json:"head,omitempty"`
	Detail  string        `json:"detail,omitempty"`
}

// Run checks each project in turn. It is a pure loop over Inspect — it
// takes no store handle because a filesystem/git check needs nothing from
// the database beyond the model.Project values the caller already loaded.
func Run(projects []model.Project) []Result {
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

	branch, err := gitOutput(p.RepoPath, "rev-parse", "--abbrev-ref", "HEAD")
	if err != nil {
		return checkFailedResult(res, p.RepoPath, err)
	}
	head, err := gitOutput(p.RepoPath, "rev-parse", "HEAD")
	if err != nil {
		return checkFailedResult(res, p.RepoPath, err)
	}
	res.Branch = branch
	res.Head = head

	if p.GitRemote != "" {
		actual, err := gitOutput(p.RepoPath, "remote", "get-url", "origin")
		if err != nil {
			return checkFailedResult(res, p.RepoPath, err)
		}
		if actual != "" && actual != p.GitRemote {
			res.State = StateRemoteChanged
			res.Detail = fmt.Sprintf("recorded %s, found %s", p.GitRemote, actual)
			return res
		}
	}

	res.State = StateOK
	return res
}

// checkFailedResult reports that git could not be run in dir at all — as
// opposed to running and exiting non-zero, which gitOutput folds into a
// plain empty string because it is a normal outcome (no such ref, no such
// remote, no commits yet).
func checkFailedResult(res Result, dir string, err error) Result {
	res.State = StateCheckFailed
	res.Detail = fmt.Sprintf("could not run git in %s: %v", dir, err)
	return res
}

// gitOutput runs a git command in dir with a bounded timeout and returns
// its trimmed output.
//
// It returns a non-nil error only when git could not be run at all: the
// binary is missing from PATH, or the command did not finish within
// gitTimeout. A command that ran to completion but exited non-zero (no
// such ref, no such remote, a repository with no commits) returns ("",
// nil) — that is a normal negative result, not an environment problem,
// and callers are expected to treat the empty string accordingly.
func gitOutput(dir string, args ...string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), gitTimeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, "git", args...)
	cmd.Dir = dir
	out, err := cmd.Output()
	if err != nil {
		var execErr *exec.Error
		if errors.As(err, &execErr) {
			return "", err
		}
		if ctx.Err() != nil {
			return "", ctx.Err()
		}
		return "", nil
	}
	return strings.TrimSpace(string(out)), nil
}
