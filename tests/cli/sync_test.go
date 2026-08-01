package cli_test

import (
	"os/exec"
	"strings"
	"testing"
)

// initRepo makes a git repository with one commit and returns its path.
func initRepo(t *testing.T) string {
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
	return dir
}

func TestSyncReportsHealthyProject(t *testing.T) {
	db := newDB(t)
	repo := initRepo(t)

	mk(t, db, "project", "add", "--name", "my-app", "--path", repo)

	r := mk(t, db, "--json", "sync")
	requireCode(t, r, 0)

	var results []struct {
		State  string `json:"state"`
		Branch string `json:"branch"`
		Head   string `json:"head"`
	}
	decode(t, r, &results)

	if len(results) != 1 {
		t.Fatalf("results = %+v, want 1", results)
	}
	if results[0].State != "ok" {
		t.Errorf("state = %q, want ok", results[0].State)
	}
	if results[0].Head == "" {
		t.Error("head not reported")
	}
}

func TestSyncReportsMissingPath(t *testing.T) {
	db := newDB(t)

	mk(t, db, "project", "add", "--name", "gone", "--path", "/tmp/definitely-not-here-99")

	r := mk(t, db, "--json", "sync")
	requireCode(t, r, 0)

	var results []struct {
		State string `json:"state"`
	}
	decode(t, r, &results)
	if len(results) != 1 || results[0].State != "missing" {
		t.Errorf("results = %+v, want one missing", results)
	}
}

func TestSyncReportsNotGit(t *testing.T) {
	db := newDB(t)

	mk(t, db, "project", "add", "--name", "plain", "--path", t.TempDir())

	r := mk(t, db, "--json", "sync")
	var results []struct {
		State string `json:"state"`
	}
	decode(t, r, &results)

	if len(results) != 1 || results[0].State != "not-git" {
		t.Errorf("results = %+v, want one not-git", results)
	}
}

func TestSyncScopedByProjectFlag(t *testing.T) {
	db := newDB(t)

	mk(t, db, "project", "add", "--name", "one", "--path", initRepo(t))
	mk(t, db, "project", "add", "--name", "two", "--path", initRepo(t))

	r := mk(t, db, "--json", "-p", "one", "sync")
	requireCode(t, r, 0)

	var results []struct {
		Project struct {
			Name string `json:"name"`
		} `json:"project"`
	}
	decode(t, r, &results)

	if len(results) != 1 || results[0].Project.Name != "one" {
		t.Errorf("results = %+v, want just one", results)
	}
}

// TestSyncScopedByProjectFlagUnknownProject checks that -p with an absent
// project fails the same way every other -p-scoped command does: exit 1,
// not a silently empty or successful sync. This is the defect class this
// project keeps hitting — a scoped variant skipping a check its unscoped
// sibling never needed.
func TestSyncScopedByProjectFlagUnknownProject(t *testing.T) {
	db := newDB(t)

	r := mk(t, db, "-p", "nope99", "sync")
	requireCode(t, r, 1)
}

func TestSyncPlainOutputNamesDrift(t *testing.T) {
	db := newDB(t)

	mk(t, db, "project", "add", "--name", "gone", "--path", "/tmp/definitely-not-here-99")

	r := mk(t, db, "sync")
	requireCode(t, r, 0)

	if !strings.Contains(r.stdout, "gone") || !strings.Contains(r.stdout, "missing") {
		t.Errorf("plain output = %q, want the project and its state", r.stdout)
	}
}

func TestSyncNoProjects(t *testing.T) {
	r := mk(t, newDB(t), "--json", "sync")
	requireCode(t, r, 0)

	if strings.TrimSpace(r.stdout) != "[]" {
		t.Errorf("stdout = %q, want []", r.stdout)
	}
}
