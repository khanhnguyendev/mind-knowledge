package cli_test

import (
	"strings"
	"testing"
)

func TestProjectAddPrintsID(t *testing.T) {
	db := newDB(t)

	r := mk(t, db, "project", "add", "--name", "my-app", "--path", "/tmp/my-app")
	requireCode(t, r, 0)

	id := strings.TrimSpace(r.stdout)
	if len(id) != 6 {
		t.Fatalf("stdout = %q, want a bare 6-character id", r.stdout)
	}
}

func TestProjectAddJSONReturnsFullRecord(t *testing.T) {
	db := newDB(t)

	r := mk(t, db, "--json", "project", "add",
		"--name", "my-app", "--path", "/tmp/my-app", "--remote", "git@github.com:me/my-app.git")
	requireCode(t, r, 0)

	var p struct {
		ID        string `json:"id"`
		Name      string `json:"name"`
		RepoPath  string `json:"repo_path"`
		GitRemote string `json:"git_remote"`
		Status    string `json:"status"`
	}
	decode(t, r, &p)

	if p.Name != "my-app" || p.RepoPath != "/tmp/my-app" {
		t.Errorf("record = %+v", p)
	}
	if p.GitRemote != "git@github.com:me/my-app.git" {
		t.Errorf("git_remote = %q", p.GitRemote)
	}
	if p.Status != "active" {
		t.Errorf("status = %q, want active", p.Status)
	}
}

func TestProjectAddDuplicateExitsTwo(t *testing.T) {
	db := newDB(t)

	mk(t, db, "project", "add", "--name", "my-app", "--path", "/tmp/a")
	r := mk(t, db, "project", "add", "--name", "my-app", "--path", "/tmp/b")
	requireCode(t, r, 2)
}

func TestProjectViewMissingExitsOne(t *testing.T) {
	db := newDB(t)

	r := mk(t, db, "project", "view", "nope99")
	requireCode(t, r, 1)
}

func TestProjectListJSONIsArray(t *testing.T) {
	db := newDB(t)

	mk(t, db, "project", "add", "--name", "alpha", "--path", "/tmp/alpha")
	mk(t, db, "project", "add", "--name", "beta", "--path", "/tmp/beta")

	r := mk(t, db, "--json", "project", "ls")
	requireCode(t, r, 0)

	var projects []struct {
		Name string `json:"name"`
	}
	decode(t, r, &projects)
	if len(projects) != 2 {
		t.Fatalf("got %d projects, want 2: %s", len(projects), r.stdout)
	}
	if projects[0].Name != "alpha" || projects[1].Name != "beta" {
		t.Errorf("projects not ordered by name: %+v", projects)
	}
}

func TestProjectListEmptyIsEmptyJSONArray(t *testing.T) {
	r := mk(t, newDB(t), "--json", "project", "ls")
	requireCode(t, r, 0)

	if strings.TrimSpace(r.stdout) != "[]" {
		t.Errorf("stdout = %q, want []", r.stdout)
	}
}

func TestProjectEditChangesStatus(t *testing.T) {
	db := newDB(t)

	add := mk(t, db, "project", "add", "--name", "my-app", "--path", "/tmp/a")
	id := strings.TrimSpace(add.stdout)

	r := mk(t, db, "--json", "project", "edit", id, "--status", "archived")
	requireCode(t, r, 0)

	var p struct {
		Status string `json:"status"`
	}
	decode(t, r, &p)
	if p.Status != "archived" {
		t.Errorf("status = %q, want archived", p.Status)
	}
}

func TestProjectEditRejectsUnknownStatus(t *testing.T) {
	db := newDB(t)

	add := mk(t, db, "project", "add", "--name", "my-app", "--path", "/tmp/a")
	id := strings.TrimSpace(add.stdout)

	r := mk(t, db, "project", "edit", id, "--status", "retired")
	requireCode(t, r, 2)
}

func TestProjectRemove(t *testing.T) {
	db := newDB(t)

	add := mk(t, db, "project", "add", "--name", "my-app", "--path", "/tmp/a")
	id := strings.TrimSpace(add.stdout)

	requireCode(t, mk(t, db, "project", "rm", id), 0)
	requireCode(t, mk(t, db, "project", "view", id), 1)
}
