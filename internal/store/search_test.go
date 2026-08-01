package store

import (
	"database/sql"
	"path/filepath"
	"strings"
	"testing"
)

func TestSearchFindsStoryByTitle(t *testing.T) {
	s := testStore(t)
	eid := seedEpic(t, s)
	st, _ := s.CreateStory(eid, "add login endpoint", "")

	hits, err := s.Search("login", nil, 0)
	if err != nil {
		t.Fatalf("Search: %v", err)
	}
	if len(hits) != 1 {
		t.Fatalf("hits = %+v, want 1", hits)
	}
	if hits[0].ID != st.ID || hits[0].Kind != "story" {
		t.Errorf("hit = %+v, want story %s", hits[0], st.ID)
	}
}

func TestSearchFindsWikiPageByBody(t *testing.T) {
	s := testStore(t)
	s.CreateWikiPage("", "Auth Model", "concept", "", "sessions expire after 30 minutes", "")

	hits, err := s.Search("expire", nil, 0)
	if err != nil {
		t.Fatalf("Search: %v", err)
	}
	if len(hits) != 1 || hits[0].Kind != "wiki" {
		t.Fatalf("hits = %+v, want one wiki hit", hits)
	}
	if !strings.Contains(strings.ToLower(hits[0].Snippet), "expire") {
		t.Errorf("snippet = %q, want the matched term", hits[0].Snippet)
	}
}

func TestSearchFindsSourceByBody(t *testing.T) {
	s := testStore(t)
	s.CreateSource("", "An article", "article", "the memex was described in 1945", "")

	hits, _ := s.Search("memex", nil, 0)
	if len(hits) != 1 || hits[0].Kind != "source" {
		t.Errorf("hits = %+v, want one source hit", hits)
	}
}

func TestSearchFiltersByKind(t *testing.T) {
	s := testStore(t)
	eid := seedEpic(t, s)
	s.CreateStory(eid, "auth work", "")
	s.CreateWikiPage("", "Auth Model", "concept", "", "auth work", "")

	all, _ := s.Search("auth", nil, 0)
	if len(all) != 2 {
		t.Fatalf("unfiltered hits = %+v, want 2", all)
	}

	wikiOnly, _ := s.Search("auth", []string{"wiki"}, 0)
	if len(wikiOnly) != 1 || wikiOnly[0].Kind != "wiki" {
		t.Errorf("wiki hits = %+v", wikiOnly)
	}
}

func TestSearchTracksUpdatesAndDeletes(t *testing.T) {
	s := testStore(t)
	eid := seedEpic(t, s)
	st, _ := s.CreateStory(eid, "original wording", "")

	newTitle := "replaced wording"
	if _, err := s.UpdateStory(st.ID, StoryFields{Title: &newTitle}); err != nil {
		t.Fatalf("UpdateStory: %v", err)
	}

	if hits, _ := s.Search("original", nil, 0); len(hits) != 0 {
		t.Errorf("stale index still matches the old title: %+v", hits)
	}
	if hits, _ := s.Search("replaced", nil, 0); len(hits) != 1 {
		t.Errorf("index missed the new title: %+v", hits)
	}

	if err := s.DeleteStory(st.ID); err != nil {
		t.Fatalf("DeleteStory: %v", err)
	}
	if hits, _ := s.Search("replaced", nil, 0); len(hits) != 0 {
		t.Errorf("deleted story still indexed: %+v", hits)
	}
}

func TestSearchLimits(t *testing.T) {
	s := testStore(t)
	eid := seedEpic(t, s)
	for _, title := range []string{"auth one", "auth two", "auth three"} {
		s.CreateStory(eid, title, "")
	}

	hits, _ := s.Search("auth", nil, 2)
	if len(hits) != 2 {
		t.Errorf("hits = %d, want 2", len(hits))
	}
}

func TestSearchEmptyQueryIsInvalid(t *testing.T) {
	s := testStore(t)

	if _, err := s.Search("   ", nil, 0); err == nil {
		t.Error("empty query accepted, want ErrInvalid")
	}
}

func TestSearchSpecialCharactersDoNotCrash(t *testing.T) {
	s := testStore(t)
	eid := seedEpic(t, s)
	s.CreateStory(eid, "auth work", "")

	// FTS5 treats several characters as operators. A user query must never
	// surface a syntax error from SQLite.
	for _, q := range []string{`auth"`, `auth(`, `"unclosed`, `auth OR`, `*`} {
		if _, err := s.Search(q, nil, 0); err != nil {
			t.Errorf("Search(%q) returned %v, want a clean result or ErrInvalid", q, err)
		}
	}
}

// TestSearchBackfillsPreexistingData reproduces the exact situation
// migration 0002's backfill exists for: rows written to a database that
// only had migration 0001 applied, before search_index or its triggers
// existed. Store.Open always runs every pending migration in one call, so
// the only way to observe "0001 applied, 0002 not yet" is to build that
// database by hand — applying 0001 and recording its version directly,
// bypassing Open — then call Open and confirm the backfill in 0002 makes
// the already-present rows searchable.
func TestSearchBackfillsPreexistingData(t *testing.T) {
	path := filepath.Join(t.TempDir(), "test.db")

	raw, err := sql.Open("sqlite", path)
	if err != nil {
		t.Fatalf("sql.Open: %v", err)
	}

	body, err := migrationFS.ReadFile("migrations/0001_init.sql")
	if err != nil {
		t.Fatalf("reading 0001_init.sql: %v", err)
	}
	if _, err := raw.Exec(string(body)); err != nil {
		t.Fatalf("applying 0001_init.sql: %v", err)
	}
	if _, err := raw.Exec(
		`CREATE TABLE schema_migrations (version INTEGER PRIMARY KEY)`); err != nil {
		t.Fatalf("creating schema_migrations: %v", err)
	}
	if _, err := raw.Exec(`INSERT INTO schema_migrations (version) VALUES (1)`); err != nil {
		t.Fatalf("recording version 1: %v", err)
	}

	now := "2026-01-01T00:00:00Z"
	if _, err := raw.Exec(
		`INSERT INTO projects (id, name, repo_path, created_at, updated_at)
		 VALUES ('p1', 'proj', '/tmp/proj', ?, ?)`, now, now); err != nil {
		t.Fatalf("inserting project: %v", err)
	}
	if _, err := raw.Exec(
		`INSERT INTO epics (id, project_id, title, position, created_at, updated_at)
		 VALUES ('e1', 'p1', 'Epic', 1, ?, ?)`, now, now); err != nil {
		t.Fatalf("inserting epic: %v", err)
	}
	if _, err := raw.Exec(
		`INSERT INTO stories (id, epic_id, title, position, created_at, updated_at)
		 VALUES ('s1', 'e1', 'backfilled story wombat', 1, ?, ?)`, now, now); err != nil {
		t.Fatalf("inserting story: %v", err)
	}
	if _, err := raw.Exec(
		`INSERT INTO wiki_pages (id, slug, title, body, created_at, updated_at)
		 VALUES ('w1', 'backfilled-wiki', 'wiki title', 'backfilled wiki narwhal', ?, ?)`,
		now, now); err != nil {
		t.Fatalf("inserting wiki page: %v", err)
	}
	if _, err := raw.Exec(
		`INSERT INTO sources (id, title, body, ingested_at)
		 VALUES ('src1', 'source title', 'backfilled source platypus', ?)`, now); err != nil {
		t.Fatalf("inserting source: %v", err)
	}
	if err := raw.Close(); err != nil {
		t.Fatalf("closing raw handle: %v", err)
	}

	// Open now finds schema version 1 and applies only migration 0002,
	// whose trailing INSERT ... SELECT statements are the code under test.
	s, err := Open(path)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer s.Close()

	var version int
	if err := s.db.QueryRow(
		`SELECT MAX(version) FROM schema_migrations`).Scan(&version); err != nil {
		t.Fatalf("reading schema_migrations: %v", err)
	}
	if version != 2 {
		t.Fatalf("schema version = %d, want 2 (0002 should have applied)", version)
	}

	for term, wantKind := range map[string]string{
		"wombat":   "story",
		"narwhal":  "wiki",
		"platypus": "source",
	} {
		hits, err := s.Search(term, nil, 0)
		if err != nil {
			t.Fatalf("Search(%q): %v", term, err)
		}
		if len(hits) != 1 || hits[0].Kind != wantKind {
			t.Errorf("Search(%q) = %+v, want one %s hit (backfill missed pre-existing data)",
				term, hits, wantKind)
		}
	}
}
