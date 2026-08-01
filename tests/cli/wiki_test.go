package cli_test

import (
	"strings"
	"testing"
)

func TestWikiAddDerivesSlug(t *testing.T) {
	db := newDB(t)

	r := mk(t, db, "--json", "wiki", "add",
		"--title", "Auth Model", "--kind", "concept",
		"--summary", "How auth works", "--body", "Body.")
	requireCode(t, r, 0)

	var p struct {
		Slug    string `json:"slug"`
		Kind    string `json:"kind"`
		Status  string `json:"status"`
		Summary string `json:"summary"`
	}
	decode(t, r, &p)

	if p.Slug != "auth-model" {
		t.Errorf("slug = %q, want auth-model", p.Slug)
	}
	if p.Status != "current" {
		t.Errorf("status = %q, want current", p.Status)
	}
	if p.Summary != "How auth works" {
		t.Errorf("summary = %q", p.Summary)
	}
}

func TestWikiViewBySlug(t *testing.T) {
	db := newDB(t)

	mk(t, db, "wiki", "add", "--title", "Auth Model", "--body", "The body text.")

	r := mk(t, db, "wiki", "view", "auth-model")
	requireCode(t, r, 0)
	if !strings.Contains(r.stdout, "The body text.") {
		t.Errorf("view = %q, want the body", r.stdout)
	}
}

func TestWikiAddDuplicateSlugExitsTwo(t *testing.T) {
	db := newDB(t)

	mk(t, db, "wiki", "add", "--title", "Auth Model", "--body", "a")
	r := mk(t, db, "wiki", "add", "--title", "Auth Model", "--body", "b")
	requireCode(t, r, 2)
}

func TestWikiEditMarksSuperseded(t *testing.T) {
	db := newDB(t)

	mk(t, db, "wiki", "add", "--title", "Auth Model", "--body", "a")

	r := mk(t, db, "--json", "wiki", "edit", "auth-model", "--status", "superseded")
	requireCode(t, r, 0)

	var p struct {
		Status string `json:"status"`
	}
	decode(t, r, &p)
	if p.Status != "superseded" {
		t.Errorf("status = %q, want superseded", p.Status)
	}
}

func TestWikiListFiltersByKind(t *testing.T) {
	db := newDB(t)

	mk(t, db, "wiki", "add", "--title", "A Concept", "--kind", "concept", "--body", "a")
	mk(t, db, "wiki", "add", "--title", "A Spec", "--kind", "spec", "--body", "b")

	r := mk(t, db, "--json", "wiki", "ls", "--kind", "spec")
	requireCode(t, r, 0)

	var pages []struct {
		Title string `json:"title"`
	}
	decode(t, r, &pages)
	if len(pages) != 1 || pages[0].Title != "A Spec" {
		t.Errorf("pages = %+v, want just A Spec", pages)
	}
}

func TestWikiScopedToProject(t *testing.T) {
	db := newDB(t)
	pid := seedProject(t, db)

	mk(t, db, "wiki", "add", "--title", "Scoped", "--kind", "spec",
		"--body", "a", "--project", pid)
	mk(t, db, "wiki", "add", "--title", "Global", "--kind", "concept", "--body", "b")

	r := mk(t, db, "--json", "-p", pid, "wiki", "ls")
	requireCode(t, r, 0)

	var pages []struct {
		Title string `json:"title"`
	}
	decode(t, r, &pages)
	if len(pages) != 1 || pages[0].Title != "Scoped" {
		t.Errorf("pages = %+v, want just Scoped", pages)
	}
}

func TestWikiRemove(t *testing.T) {
	db := newDB(t)

	mk(t, db, "wiki", "add", "--title", "Auth Model", "--body", "a")
	requireCode(t, mk(t, db, "wiki", "rm", "auth-model"), 0)
	requireCode(t, mk(t, db, "wiki", "view", "auth-model"), 1)
}

// TestWikiListEmptyIsJSONArray guards against the classic Go null-vs-[]
// bug: json.Unmarshal treats "null" and "[]" identically, so a test that
// decodes into a slice and checks len() == 0 would pass whether ls printed
// "[]" or "null". Asserting on the raw stdout text is the only way to
// actually pin the contract that empty result sets serialize as [].
func TestWikiListEmptyIsJSONArray(t *testing.T) {
	db := newDB(t)

	r := mk(t, db, "--json", "wiki", "ls")
	requireCode(t, r, 0)

	got := strings.TrimSpace(r.stdout)
	if got != "[]" {
		t.Errorf("stdout = %q, want the literal text \"[]\"", got)
	}
}
