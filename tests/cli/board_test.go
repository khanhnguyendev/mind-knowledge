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

func TestBoardJSONEmptyEpicsSerializesToEmptyArray(t *testing.T) {
	db := newDB(t)
	seedProject(t, db)

	r := mk(t, db, "--json", "board")
	requireCode(t, r, 0)

	if !strings.Contains(r.stdout, "\"epics\":[]") {
		t.Errorf("empty epics must serialize to [] not null:\n%s", r.stdout)
	}
}

func TestBoardJSONEmptyStoriesSerializesToEmptyArray(t *testing.T) {
	db := newDB(t)

	pid := seedProject(t, db)
	_ = strings.TrimSpace(mk(t, db, "epic", "create", "--project", pid, "--title", "Auth").stdout)

	r := mk(t, db, "--json", "board")
	requireCode(t, r, 0)

	if !strings.Contains(r.stdout, "\"stories\":[]") {
		t.Errorf("empty stories must serialize to [] not null:\n%s", r.stdout)
	}
}

func TestBoardStatusFilterShowsOnlyMatchingStories(t *testing.T) {
	db := newDB(t)

	pid := seedProject(t, db)
	eid1 := strings.TrimSpace(mk(t, db, "epic", "create", "--project", pid, "--title", "Auth").stdout)
	eid2 := strings.TrimSpace(mk(t, db, "epic", "create", "--project", pid, "--title", "API").stdout)

	// Epic 1: one ready story, one review story
	ready := strings.TrimSpace(mk(t, db, "story", "create", "--epic", eid1, "--title", "add login").stdout)
	requireCode(t, mk(t, db, "story", "mv", ready, "--to", "ready"), 0)

	review := strings.TrimSpace(mk(t, db, "story", "create", "--epic", eid1, "--title", "migrate hashes").stdout)
	requireCode(t, mk(t, db, "story", "mv", review, "--to", "review"), 0)

	// Epic 2: one review story
	review2 := strings.TrimSpace(mk(t, db, "story", "create", "--epic", eid2, "--title", "docs").stdout)
	requireCode(t, mk(t, db, "story", "mv", review2, "--to", "review"), 0)

	r := mk(t, db, "--json", "board", "--status", "ready")
	requireCode(t, r, 0)

	var board []struct {
		Epics []struct {
			Title  string `json:"title"`
			Stories []struct {
				Title string `json:"title"`
			} `json:"stories"`
		} `json:"epics"`
	}
	decode(t, r, &board)

	if len(board) != 1 {
		t.Fatalf("expected 1 project, got %d", len(board))
	}
	if len(board[0].Epics) != 2 {
		t.Fatalf("expected 2 epics, got %d", len(board[0].Epics))
	}

	// First epic should have the ready story
	if len(board[0].Epics[0].Stories) != 1 {
		t.Errorf("expected 1 story in first epic, got %d", len(board[0].Epics[0].Stories))
	}
	if board[0].Epics[0].Stories[0].Title != "add login" {
		t.Errorf("wrong story in first epic: %s", board[0].Epics[0].Stories[0].Title)
	}

	// Second epic should have no stories (review stories filtered out)
	if len(board[0].Epics[1].Stories) != 0 {
		t.Errorf("expected 0 stories in second epic, got %d: %+v",
			len(board[0].Epics[1].Stories), board[0].Epics[1].Stories)
	}

	if !strings.Contains(r.stdout, "\"stories\":[]") {
		t.Errorf("epic with filtered-out stories must serialize empty array:\n%s", r.stdout)
	}
}

func TestBoardProjectFlagShowsArchivedWithoutAll(t *testing.T) {
	db := newDB(t)

	seedProject(t, db)
	archived := mk(t, db, "project", "add", "--name", "old-thing", "--path", "/tmp/old")
	aid := strings.TrimSpace(archived.stdout)
	requireCode(t, mk(t, db, "project", "edit", aid, "--status", "archived"), 0)

	r := mk(t, db, "-p", "old-thing", "board")
	requireCode(t, r, 0)

	if !strings.Contains(r.stdout, "old-thing") {
		t.Errorf("-p flag must show archived project even without --all:\n%s", r.stdout)
	}
}
