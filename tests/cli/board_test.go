package cli_test

import (
	"strings"
	"testing"
)

func TestBoardShowsActiveProjectsOnly(t *testing.T) {
	db := newDB(t)

	pid := seedProject(t, db)
	archived := mk(t, db, "project", "add", "--name", "old-thing", "--path", "/tmp/old")
	aid := strings.TrimSpace(archived.stdout)
	requireCode(t, mk(t, db, "project", "edit", aid, "--status", "archived"), 0)

	eid := strings.TrimSpace(mk(t, db, "epic", "create", "--project", pid, "--title", "Auth").stdout)
	mk(t, db, "story", "create", "--epic", eid, "--title", "add login endpoint")

	r := mk(t, db, "board")
	requireCode(t, r, 0)

	if !strings.Contains(r.stdout, "my-app") {
		t.Errorf("board missing active project:\n%s", r.stdout)
	}
	if strings.Contains(r.stdout, "old-thing") {
		t.Errorf("board included archived project:\n%s", r.stdout)
	}
	if !strings.Contains(r.stdout, "add login endpoint") {
		t.Errorf("board missing story:\n%s", r.stdout)
	}
}

func TestBoardAllIncludesArchived(t *testing.T) {
	db := newDB(t)

	seedProject(t, db)
	archived := mk(t, db, "project", "add", "--name", "old-thing", "--path", "/tmp/old")
	aid := strings.TrimSpace(archived.stdout)
	requireCode(t, mk(t, db, "project", "edit", aid, "--status", "archived"), 0)

	r := mk(t, db, "board", "--all")
	requireCode(t, r, 0)

	if !strings.Contains(r.stdout, "old-thing") {
		t.Errorf("--all board missing archived project:\n%s", r.stdout)
	}
}

func TestBoardScopedByProjectFlag(t *testing.T) {
	db := newDB(t)

	seedProject(t, db)
	mk(t, db, "project", "add", "--name", "other", "--path", "/tmp/other")

	r := mk(t, db, "-p", "other", "board")
	requireCode(t, r, 0)

	if strings.Contains(r.stdout, "my-app") {
		t.Errorf("scoped board leaked another project:\n%s", r.stdout)
	}
	if !strings.Contains(r.stdout, "other") {
		t.Errorf("scoped board missing its project:\n%s", r.stdout)
	}
}

func TestBoardJSONNestsStoriesUnderEpics(t *testing.T) {
	db := newDB(t)

	pid := seedProject(t, db)
	eid := strings.TrimSpace(mk(t, db, "epic", "create", "--project", pid, "--title", "Auth").stdout)
	mk(t, db, "story", "create", "--epic", eid, "--title", "add login endpoint")

	r := mk(t, db, "--json", "board")
	requireCode(t, r, 0)

	var board []struct {
		Name  string `json:"name"`
		Epics []struct {
			ID      string `json:"id"`
			Title   string `json:"title"`
			Stories []struct {
				Title  string `json:"title"`
				Status string `json:"status"`
			} `json:"stories"`
		} `json:"epics"`
	}
	decode(t, r, &board)

	if len(board) != 1 || board[0].Name != "my-app" {
		t.Fatalf("board = %+v", board)
	}
	if len(board[0].Epics) != 1 || board[0].Epics[0].ID != eid {
		t.Fatalf("epics = %+v", board[0].Epics)
	}
	if len(board[0].Epics[0].Stories) != 1 {
		t.Fatalf("stories = %+v", board[0].Epics[0].Stories)
	}
	if board[0].Epics[0].Stories[0].Title != "add login endpoint" {
		t.Errorf("story = %+v", board[0].Epics[0].Stories[0])
	}
}

func TestBoardEmptyDatabase(t *testing.T) {
	r := mk(t, newDB(t), "board")
	requireCode(t, r, 0)
	if strings.TrimSpace(r.stdout) == "" {
		t.Error("empty board printed nothing")
	}
}
