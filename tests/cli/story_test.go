package cli_test

import (
	"strings"
	"testing"
)

// seedEpic registers a project and an epic, returning the epic id.
func seedEpic(t *testing.T, db string) string {
	t.Helper()
	pid := seedProject(t, db)
	r := mk(t, db, "epic", "create", "--project", pid, "--title", "Auth overhaul")
	requireCode(t, r, 0)
	return strings.TrimSpace(r.stdout)
}

func TestStoryCreateAndView(t *testing.T) {
	db := newDB(t)
	eid := seedEpic(t, db)

	create := mk(t, db, "story", "create", "--epic", eid, "--title", "add login endpoint")
	requireCode(t, create, 0)
	sid := strings.TrimSpace(create.stdout)

	r := mk(t, db, "--json", "story", "view", sid)
	requireCode(t, r, 0)

	var st struct {
		ID       string `json:"id"`
		EpicID   string `json:"epic_id"`
		Title    string `json:"title"`
		Status   string `json:"status"`
		Priority string `json:"priority"`
	}
	decode(t, r, &st)
	if st.ID != sid || st.EpicID != eid {
		t.Errorf("story = %+v", st)
	}
	if st.Status != "backlog" || st.Priority != "med" {
		t.Errorf("defaults wrong: %+v", st)
	}
}

func TestStoryEditSetsPlan(t *testing.T) {
	db := newDB(t)
	eid := seedEpic(t, db)
	sid := strings.TrimSpace(mk(t, db, "story", "create", "--epic", eid, "--title", "x").stdout)

	plan := "1. write failing test\n2. implement\n3. commit"
	requireCode(t, mk(t, db, "story", "edit", sid, "--plan", plan), 0)

	r := mk(t, db, "--json", "story", "view", sid)
	requireCode(t, r, 0)

	var st struct {
		Plan string `json:"plan"`
	}
	decode(t, r, &st)
	if st.Plan != plan {
		t.Errorf("plan = %q, want %q", st.Plan, plan)
	}
}

func TestStoryAppendNotesAccumulates(t *testing.T) {
	db := newDB(t)
	eid := seedEpic(t, db)
	sid := strings.TrimSpace(mk(t, db, "story", "create", "--epic", eid, "--title", "x").stdout)

	requireCode(t, mk(t, db, "story", "edit", sid, "--append-notes", "first"), 0)
	requireCode(t, mk(t, db, "story", "edit", sid, "--append-notes", "second"), 0)

	r := mk(t, db, "--json", "story", "view", sid)
	var st struct {
		Notes string `json:"notes"`
	}
	decode(t, r, &st)
	if !strings.Contains(st.Notes, "first") || !strings.Contains(st.Notes, "second") {
		t.Errorf("notes = %q, want both appends", st.Notes)
	}
}

func TestStoryMoveToDoneWithoutPlanSucceeds(t *testing.T) {
	db := newDB(t)
	eid := seedEpic(t, db)
	sid := strings.TrimSpace(mk(t, db, "story", "create", "--epic", eid, "--title", "x").stdout)

	// The dumb core must not block this. mk doctor reports it later.
	r := mk(t, db, "--json", "story", "mv", sid, "--to", "done")
	requireCode(t, r, 0)

	var st struct {
		Status string `json:"status"`
	}
	decode(t, r, &st)
	if st.Status != "done" {
		t.Errorf("status = %q, want done", st.Status)
	}
}

func TestStoryMoveRejectsUnknownStatus(t *testing.T) {
	db := newDB(t)
	eid := seedEpic(t, db)
	sid := strings.TrimSpace(mk(t, db, "story", "create", "--epic", eid, "--title", "x").stdout)

	requireCode(t, mk(t, db, "story", "mv", sid, "--to", "closed"), 2)
}

func TestStoryListFilters(t *testing.T) {
	db := newDB(t)
	eid := seedEpic(t, db)
	a := strings.TrimSpace(mk(t, db, "story", "create", "--epic", eid, "--title", "a").stdout)
	mk(t, db, "story", "create", "--epic", eid, "--title", "b")
	requireCode(t, mk(t, db, "story", "mv", a, "--to", "ready"), 0)

	r := mk(t, db, "--json", "story", "ls", "--status", "ready")
	requireCode(t, r, 0)

	var stories []struct {
		ID string `json:"id"`
	}
	decode(t, r, &stories)
	if len(stories) != 1 || stories[0].ID != a {
		t.Errorf("stories = %+v, want just %s", stories, a)
	}
}

func TestStoryRemove(t *testing.T) {
	db := newDB(t)
	eid := seedEpic(t, db)
	sid := strings.TrimSpace(mk(t, db, "story", "create", "--epic", eid, "--title", "x").stdout)

	requireCode(t, mk(t, db, "story", "rm", sid), 0)
	requireCode(t, mk(t, db, "story", "view", sid), 1)
}
