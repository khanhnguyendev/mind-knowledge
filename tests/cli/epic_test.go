package cli_test

import (
	"strings"
	"testing"
)

// seedProject registers a project and returns its id.
func seedProject(t *testing.T, db string) string {
	t.Helper()
	r := mk(t, db, "project", "add", "--name", "my-app", "--path", "/tmp/my-app")
	requireCode(t, r, 0)
	return strings.TrimSpace(r.stdout)
}

func TestEpicCreateAndView(t *testing.T) {
	db := newDB(t)
	pid := seedProject(t, db)

	create := mk(t, db, "epic", "create", "--project", pid, "--title", "Auth overhaul")
	requireCode(t, create, 0)
	eid := strings.TrimSpace(create.stdout)

	r := mk(t, db, "--json", "epic", "view", eid)
	requireCode(t, r, 0)

	var e struct {
		ID        string `json:"id"`
		ProjectID string `json:"project_id"`
		Title     string `json:"title"`
		Status    string `json:"status"`
	}
	decode(t, r, &e)
	if e.ID != eid || e.ProjectID != pid {
		t.Errorf("epic = %+v, want id %q in project %q", e, eid, pid)
	}
	if e.Title != "Auth overhaul" || e.Status != "backlog" {
		t.Errorf("epic = %+v", e)
	}
}

func TestEpicCreateUnknownProjectExitsOne(t *testing.T) {
	db := newDB(t)

	r := mk(t, db, "epic", "create", "--project", "nope99", "--title", "X")
	requireCode(t, r, 1)
}

func TestEpicListScopedByProjectFlag(t *testing.T) {
	db := newDB(t)
	pid := seedProject(t, db)

	other := mk(t, db, "project", "add", "--name", "other", "--path", "/tmp/other")
	oid := strings.TrimSpace(other.stdout)

	mk(t, db, "epic", "create", "--project", pid, "--title", "Mine")
	mk(t, db, "epic", "create", "--project", oid, "--title", "Theirs")

	r := mk(t, db, "--json", "-p", pid, "epic", "ls")
	requireCode(t, r, 0)

	var epics []struct {
		Title string `json:"title"`
	}
	decode(t, r, &epics)
	if len(epics) != 1 || epics[0].Title != "Mine" {
		t.Errorf("epics = %+v, want just Mine", epics)
	}
}

func TestEpicMoveStatus(t *testing.T) {
	db := newDB(t)
	pid := seedProject(t, db)
	eid := strings.TrimSpace(mk(t, db, "epic", "create", "--project", pid, "--title", "X").stdout)

	r := mk(t, db, "--json", "epic", "mv", eid, "--to", "in-progress")
	requireCode(t, r, 0)

	var e struct {
		Status string `json:"status"`
	}
	decode(t, r, &e)
	if e.Status != "in-progress" {
		t.Errorf("status = %q, want in-progress", e.Status)
	}
}

func TestEpicMoveRejectsStoryStatus(t *testing.T) {
	db := newDB(t)
	pid := seedProject(t, db)
	eid := strings.TrimSpace(mk(t, db, "epic", "create", "--project", pid, "--title", "X").stdout)

	r := mk(t, db, "epic", "mv", eid, "--to", "review")
	requireCode(t, r, 2)
}

func TestEpicMovePosition(t *testing.T) {
	db := newDB(t)
	pid := seedProject(t, db)
	eid := strings.TrimSpace(mk(t, db, "epic", "create", "--project", pid, "--title", "X").stdout)

	r := mk(t, db, "--json", "epic", "mv", eid, "--pos", "7")
	requireCode(t, r, 0)

	var e struct {
		Position int `json:"position"`
	}
	decode(t, r, &e)
	if e.Position != 7 {
		t.Errorf("position = %d, want 7", e.Position)
	}
}

func TestEpicMoveRequiresOneFlag(t *testing.T) {
	db := newDB(t)
	pid := seedProject(t, db)
	eid := strings.TrimSpace(mk(t, db, "epic", "create", "--project", pid, "--title", "X").stdout)

	r := mk(t, db, "epic", "mv", eid)
	requireCode(t, r, 2)
}
