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

func TestEpicEditProjectRecomputesPosition(t *testing.T) {
	db := newDB(t)
	p1 := seedProject(t, db)

	// Create second project
	p2r := mk(t, db, "project", "add", "--name", "proj2", "--path", "/tmp/proj2")
	requireCode(t, p2r, 0)
	p2 := strings.TrimSpace(p2r.stdout)

	// Create epics in proj1
	e1r := mk(t, db, "epic", "create", "--project", p1, "--title", "Epic One")
	requireCode(t, e1r, 0)
	eid := strings.TrimSpace(e1r.stdout)

	// Create existing epics in proj2
	e3r := mk(t, db, "epic", "create", "--project", p2, "--title", "Existing One")
	requireCode(t, e3r, 0)
	e4r := mk(t, db, "epic", "create", "--project", p2, "--title", "Existing Two")
	requireCode(t, e4r, 0)

	// Get the position of e4 (last epic in proj2)
	var e4 struct {
		Position int `json:"position"`
	}
	e4json := mk(t, db, "--json", "epic", "view", strings.TrimSpace(e4r.stdout))
	requireCode(t, e4json, 0)
	decode(t, e4json, &e4)

	// Move e1 to proj2 using edit. Reassignment is --set-project, not -p:
	// -p is a read-time scope everywhere else, and overloading it here
	// silently moved epics on every edit that merely passed it through.
	r := mk(t, db, "--json", "epic", "edit", eid, "--set-project", p2)
	requireCode(t, r, 0)

	// Verify the moved epic has a position greater than the old max in proj2
	var movedEpic struct {
		ProjectID string `json:"project_id"`
		Position  int    `json:"position"`
	}
	decode(t, r, &movedEpic)
	if movedEpic.ProjectID != p2 {
		t.Errorf("ProjectID = %q, want %q", movedEpic.ProjectID, p2)
	}
	if movedEpic.Position <= e4.Position {
		t.Errorf("position = %d, want > %d (should be after existing siblings)", movedEpic.Position, e4.Position)
	}
}
