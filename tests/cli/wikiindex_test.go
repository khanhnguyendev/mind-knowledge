package cli_test

import (
	"strings"
	"testing"
)

func TestWikiIndexRendersMarkdown(t *testing.T) {
	db := newDB(t)

	mk(t, db, "wiki", "add", "--title", "Auth Model", "--kind", "concept",
		"--summary", "How auth works", "--body", "a")
	mk(t, db, "wiki", "add", "--title", "mk spec", "--kind", "spec",
		"--summary", "The binary design", "--body", "b")

	r := mk(t, db, "wiki", "index")
	requireCode(t, r, 0)

	for _, want := range []string{
		"# Wiki Index", "## concept", "## spec",
		"auth-model", "How auth works", "mk-spec", "The binary design",
	} {
		if !strings.Contains(r.stdout, want) {
			t.Errorf("index missing %q:\n%s", want, r.stdout)
		}
	}
}

func TestWikiIndexJSONReturnsPages(t *testing.T) {
	db := newDB(t)

	mk(t, db, "wiki", "add", "--title", "Auth Model", "--summary", "s", "--body", "a")

	r := mk(t, db, "--json", "wiki", "index")
	requireCode(t, r, 0)

	var pages []struct {
		Slug    string `json:"slug"`
		Summary string `json:"summary"`
	}
	decode(t, r, &pages)
	if len(pages) != 1 || pages[0].Slug != "auth-model" {
		t.Errorf("pages = %+v", pages)
	}
}

func TestWikiIndexScopedToProject(t *testing.T) {
	db := newDB(t)
	pid := seedProject(t, db)

	mk(t, db, "wiki", "add", "--title", "Scoped", "--body", "a", "--project", pid)
	mk(t, db, "wiki", "add", "--title", "Global", "--body", "b")

	r := mk(t, db, "-p", pid, "wiki", "index")
	requireCode(t, r, 0)

	if strings.Contains(r.stdout, "Global") {
		t.Errorf("scoped index leaked a cross-project page:\n%s", r.stdout)
	}
}

func TestWikiIndexEmpty(t *testing.T) {
	r := mk(t, newDB(t), "wiki", "index")
	requireCode(t, r, 0)
	if !strings.Contains(r.stdout, "# Wiki Index") {
		t.Errorf("empty index = %q", r.stdout)
	}
}

func TestWikiIndexFiltersByKind(t *testing.T) {
	db := newDB(t)

	mk(t, db, "wiki", "add", "--title", "A Concept", "--kind", "concept", "--body", "a")
	mk(t, db, "wiki", "add", "--title", "A Spec", "--kind", "spec", "--body", "b")

	r := mk(t, db, "wiki", "index", "--kind", "spec")
	requireCode(t, r, 0)

	if !strings.Contains(r.stdout, "A Spec") {
		t.Errorf("spec missing from filtered index:\n%s", r.stdout)
	}
	if strings.Contains(r.stdout, "A Concept") {
		t.Errorf("concept leaked into spec-filtered index:\n%s", r.stdout)
	}
}

func TestWikiIndexFiltersByStatus(t *testing.T) {
	db := newDB(t)

	mk(t, db, "wiki", "add", "--title", "Current Page", "--body", "a")
	mk(t, db, "wiki", "add", "--title", "Superseded Page", "--body", "b")
	mk(t, db, "wiki", "edit", "superseded-page", "--status", "superseded")

	r := mk(t, db, "wiki", "index", "--status", "superseded")
	requireCode(t, r, 0)

	if !strings.Contains(r.stdout, "Superseded Page") {
		t.Errorf("superseded page missing from filtered index:\n%s", r.stdout)
	}
	if strings.Contains(r.stdout, "Current Page") {
		t.Errorf("current page leaked into superseded-filtered index:\n%s", r.stdout)
	}
}

// TestWikiIndexEmptyIsJSONArray guards against the classic Go null-vs-[]
// bug: json.Unmarshal treats "null" and "[]" identically, so a test that
// decodes into a slice and checks len() == 0 would pass whether index printed
// "[]" or "null". Asserting on the raw stdout text is the only way to
// actually pin the contract that empty result sets serialize as [].
func TestWikiIndexEmptyIsJSONArray(t *testing.T) {
	db := newDB(t)

	r := mk(t, db, "--json", "wiki", "index")
	requireCode(t, r, 0)

	got := strings.TrimSpace(r.stdout)
	if got != "[]" {
		t.Errorf("stdout = %q, want the literal text \"[]\"", got)
	}
}
