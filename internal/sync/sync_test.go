package sync

import (
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/khanhnguyendev/mind-knowledge/internal/model"
)

// initRepo creates a git repository with one commit and returns its path.
func initRepo(t *testing.T, remote string) string {
	t.Helper()
	dir := t.TempDir()

	run := func(args ...string) {
		t.Helper()
		cmd := exec.Command("git", args...)
		cmd.Dir = dir
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
	}

	run("init", "-q")
	run("config", "user.email", "test@example.com")
	run("config", "user.name", "Test")
	run("commit", "-q", "--allow-empty", "-m", "initial")
	if remote != "" {
		run("remote", "add", "origin", remote)
	}
	return dir
}

func TestInspectHealthyRepo(t *testing.T) {
	dir := initRepo(t, "git@github.com:me/my-app.git")

	got := Inspect(model.Project{
		Name:      "my-app",
		RepoPath:  dir,
		GitRemote: "git@github.com:me/my-app.git",
	})

	if got.State != StateOK {
		t.Fatalf("State = %q (%s), want ok", got.State, got.Detail)
	}
	if got.Head == "" {
		t.Error("Head not recorded")
	}
	if got.Branch == "" {
		t.Error("Branch not recorded")
	}
}

func TestInspectMissingPath(t *testing.T) {
	got := Inspect(model.Project{
		Name:     "gone",
		RepoPath: filepath.Join(t.TempDir(), "does-not-exist"),
	})

	if got.State != StateMissing {
		t.Errorf("State = %q, want missing", got.State)
	}
}

func TestInspectDirectoryWithoutGit(t *testing.T) {
	got := Inspect(model.Project{Name: "plain", RepoPath: t.TempDir()})

	if got.State != StateNotGit {
		t.Errorf("State = %q, want not-git", got.State)
	}
}

func TestInspectRemoteChanged(t *testing.T) {
	dir := initRepo(t, "git@github.com:me/actual.git")

	got := Inspect(model.Project{
		Name:      "my-app",
		RepoPath:  dir,
		GitRemote: "git@github.com:me/recorded.git",
	})

	if got.State != StateRemoteChanged {
		t.Errorf("State = %q, want remote-changed", got.State)
	}
	if got.Detail == "" {
		t.Error("Detail should name both remotes")
	}
}

func TestInspectIgnoresRemoteWhenNoneRecorded(t *testing.T) {
	dir := initRepo(t, "git@github.com:me/actual.git")

	got := Inspect(model.Project{Name: "my-app", RepoPath: dir})

	if got.State != StateOK {
		t.Errorf("State = %q, want ok when no remote was recorded", got.State)
	}
}

// TestInspectRemoteRecordedButNoOriginOnDisk guards against gitOutput's
// error-swallowing (it returns "" on any failure, including "no such
// remote") being mistaken for the remote having changed. If the repository
// has no origin at all, `git remote get-url origin` fails and gitOutput
// returns "", and that empty result must not trip the remote-changed check.
func TestInspectRemoteRecordedButNoOriginOnDisk(t *testing.T) {
	dir := initRepo(t, "") // no remote configured

	got := Inspect(model.Project{
		Name:      "my-app",
		RepoPath:  dir,
		GitRemote: "git@github.com:me/recorded.git",
	})

	if got.State != StateOK {
		t.Errorf("State = %q (%s), want ok when the repo has no origin to compare", got.State, got.Detail)
	}
}

// TestInspectEmptyRepoNoCommits covers a git repository with no commits at
// all, so HEAD has nothing to resolve to and `rev-parse HEAD` fails.
// gitOutput swallows that failure into "", so the project should still be
// reported ok (present and a real git repo) with no Head/Branch, rather
// than falling into some confusing state.
func TestInspectEmptyRepoNoCommits(t *testing.T) {
	dir := t.TempDir()
	cmd := exec.Command("git", "init", "-q")
	cmd.Dir = dir
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git init: %v\n%s", err, out)
	}

	got := Inspect(model.Project{Name: "empty", RepoPath: dir})

	if got.State != StateOK {
		t.Errorf("State = %q (%s), want ok for a repo with no commits", got.State, got.Detail)
	}
	if got.Head != "" {
		t.Errorf("Head = %q, want empty when there is no commit to resolve", got.Head)
	}
}
