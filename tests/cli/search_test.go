package cli_test

import (
	"strings"
	"testing"
)

func TestSearchAcrossKinds(t *testing.T) {
	db := newDB(t)
	eid := seedEpic(t, db)

	mk(t, db, "story", "create", "--epic", eid, "--title", "add login endpoint")
	mk(t, db, "wiki", "add", "--title", "Auth Model", "--body", "login sessions expire")
	mk(t, db, "source", "add", "--title", "An article", "--body", "about login flows")

	r := mk(t, db, "--json", "search", "login")
	requireCode(t, r, 0)

	var hits []struct {
		Kind string `json:"kind"`
		ID   string `json:"id"`
	}
	decode(t, r, &hits)
	if len(hits) != 3 {
		t.Fatalf("hits = %+v, want 3", hits)
	}

	kinds := map[string]bool{}
	for _, h := range hits {
		kinds[h.Kind] = true
	}
	for _, want := range []string{"story", "wiki", "source"} {
		if !kinds[want] {
			t.Errorf("no %s hit in %+v", want, hits)
		}
	}
}

func TestSearchFiltersByKind(t *testing.T) {
	db := newDB(t)
	eid := seedEpic(t, db)

	mk(t, db, "story", "create", "--epic", eid, "--title", "login work")
	mk(t, db, "wiki", "add", "--title", "Login Notes", "--body", "login work")

	r := mk(t, db, "--json", "search", "login", "--kind", "wiki")
	requireCode(t, r, 0)

	var hits []struct {
		Kind string `json:"kind"`
	}
	decode(t, r, &hits)
	if len(hits) != 1 || hits[0].Kind != "wiki" {
		t.Errorf("hits = %+v, want one wiki hit", hits)
	}
}

func TestSearchNoMatchesIsEmptyArray(t *testing.T) {
	db := newDB(t)

	r := mk(t, db, "--json", "search", "nothingmatchesthis")
	requireCode(t, r, 0)

	if strings.TrimSpace(r.stdout) != "[]" {
		t.Errorf("stdout = %q, want []", r.stdout)
	}
}

func TestSearchEmptyQueryExitsTwo(t *testing.T) {
	db := newDB(t)

	requireCode(t, mk(t, db, "search", "   "), 2)
}

func TestSearchUnknownKindExitsTwo(t *testing.T) {
	db := newDB(t)

	requireCode(t, mk(t, db, "search", "login", "--kind", "epic"), 2)
}

func TestSearchPlainOutputShowsSnippet(t *testing.T) {
	db := newDB(t)

	mk(t, db, "wiki", "add", "--title", "Auth Model", "--body", "sessions expire after 30 minutes")

	r := mk(t, db, "search", "expire")
	requireCode(t, r, 0)

	if !strings.Contains(r.stdout, "expire") {
		t.Errorf("plain output = %q, want the matched term", r.stdout)
	}
}

func TestSearchSurvivesQuotesInQuery(t *testing.T) {
	db := newDB(t)

	mk(t, db, "wiki", "add", "--title", "Auth Model", "--body", "a body")

	// A stray quote must not produce an FTS5 syntax error.
	r := mk(t, db, "search", `auth"`)
	requireCode(t, r, 0)
}
