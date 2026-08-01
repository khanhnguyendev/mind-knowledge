package sync

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/khanhnguyendev/mind-knowledge/internal/model"
)

// initRepo creates a git repository with one commit and returns its path.
//
// Env is set explicitly rather than inherited from the running user, so a
// developer or CI machine with e.g. commit.gpgsign=true and no available
// key, or a global core.hooksPath pointing at a failing hook, can't make
// this helper — and therefore every test that calls it — fail for reasons
// unrelated to the code under test.
func initRepo(t *testing.T, remote string) string {
	t.Helper()
	dir := t.TempDir()

	run := func(args ...string) {
		t.Helper()
		cmd := exec.Command("git", args...)
		cmd.Dir = dir
		cmd.Env = append(os.Environ(),
			"HOME="+dir,
			"GIT_CONFIG_GLOBAL=/dev/null",
			"GIT_CONFIG_SYSTEM=/dev/null")
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
	cmd.Env = append(os.Environ(),
		"HOME="+dir,
		"GIT_CONFIG_GLOBAL=/dev/null",
		"GIT_CONFIG_SYSTEM=/dev/null")
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

// TestInspectReportsCheckFailedWhenGitMissing covers gitOutput's other
// failure mode: git could not be run at all, as distinct from running and
// exiting non-zero. A machine with git absent from PATH must not render
// identically to a healthy repository with no commits — both currently
// look like an empty Head/Branch, but only one of them is actually ok.
func TestInspectReportsCheckFailedWhenGitMissing(t *testing.T) {
	dir := initRepo(t, "") // build the repo while git is still on PATH

	t.Setenv("PATH", t.TempDir()) // then take git off PATH entirely

	got := Inspect(model.Project{Name: "my-app", RepoPath: dir})

	if got.State != StateCheckFailed {
		t.Errorf("State = %q, want check-failed when git cannot be run", got.State)
	}
	if got.Detail == "" {
		t.Error("Detail should explain why the check failed")
	}
}

// TestGitOutputRespectsTimeout proves gitOutput bounds how long it waits
// on a hung git process, rather than blocking mk sync forever behind a
// stale network mount or a wedged git lock. It fakes "git" with a script
// that sleeps well past a shortened gitTimeout, so the test only waits out
// the shortened timeout — not something slow or machine-specific.
func TestGitOutputRespectsTimeout(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("fake git script assumes a POSIX shell")
	}

	binDir := t.TempDir()
	fakeGit := filepath.Join(binDir, "git")
	if err := os.WriteFile(fakeGit, []byte("#!/bin/sh\nsleep 100\n"), 0o755); err != nil {
		t.Fatalf("writing fake git: %v", err)
	}
	t.Setenv("PATH", binDir+string(os.PathListSeparator)+os.Getenv("PATH"))

	old := gitTimeout
	gitTimeout = 50 * time.Millisecond
	t.Cleanup(func() { gitTimeout = old })

	start := time.Now()
	out, err := gitOutput(t.TempDir(), "status")
	elapsed := time.Since(start)

	if err == nil {
		t.Fatalf("gitOutput returned no error for a hung command, out=%q", out)
	}
	if elapsed > 2*time.Second {
		t.Errorf("gitOutput took %s, want it to respect the shortened timeout", elapsed)
	}
}
