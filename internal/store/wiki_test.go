package store

import (
	"errors"
	"testing"
)

func TestCreateWikiPageDefaults(t *testing.T) {
	s := testStore(t)

	p, err := s.CreateWikiPage("", "Auth Model", "concept", "How auth works", "Body.", "")
	if err != nil {
		t.Fatalf("CreateWikiPage: %v", err)
	}
	if p.Slug != "auth-model" {
		t.Errorf("Slug = %q, want auth-model derived from the title", p.Slug)
	}
	if p.Status != "current" {
		t.Errorf("Status = %q, want current", p.Status)
	}
}

func TestCreateWikiPageHonorsExplicitSlug(t *testing.T) {
	s := testStore(t)

	p, err := s.CreateWikiPage("concepts/Auth Model", "Auth Model", "concept", "", "b", "")
	if err != nil {
		t.Fatalf("CreateWikiPage: %v", err)
	}
	if p.Slug != "concepts/auth-model" {
		t.Errorf("Slug = %q, want concepts/auth-model", p.Slug)
	}
}

func TestCreateWikiPageRejectsDuplicateSlug(t *testing.T) {
	s := testStore(t)

	if _, err := s.CreateWikiPage("", "Auth Model", "concept", "", "b", ""); err != nil {
		t.Fatalf("first CreateWikiPage: %v", err)
	}
	_, err := s.CreateWikiPage("", "Auth Model", "concept", "", "b", "")
	if !errors.Is(err, ErrInvalid) {
		t.Errorf("err = %v, want ErrInvalid for duplicate slug", err)
	}
}

func TestCreateWikiPageRejectsUnknownKind(t *testing.T) {
	s := testStore(t)

	_, err := s.CreateWikiPage("", "X", "article", "", "b", "")
	if !errors.Is(err, ErrInvalid) {
		t.Errorf("err = %v, want ErrInvalid (article is a source kind, not a wiki kind)", err)
	}
}

func TestGetWikiPageBySlugAndID(t *testing.T) {
	s := testStore(t)

	created, _ := s.CreateWikiPage("", "Auth Model", "concept", "", "b", "")

	bySlug, err := s.GetWikiPage("auth-model")
	if err != nil {
		t.Fatalf("GetWikiPage by slug: %v", err)
	}
	byID, err := s.GetWikiPage(created.ID)
	if err != nil {
		t.Fatalf("GetWikiPage by id: %v", err)
	}
	if bySlug.ID != created.ID || byID.Slug != created.Slug {
		t.Errorf("lookups disagree: %+v %+v", bySlug, byID)
	}
}

// TestGetWikiPageIDWinsOverSlugCollision constructs the case that bit an
// earlier task: one page's id is a string that is also a plausible slug for
// another page. GetWikiPage must resolve id before slug, deterministically,
// not by however SQLite's optimizer happens to plan a single OR query.
func TestGetWikiPageIDWinsOverSlugCollision(t *testing.T) {
	s := testStore(t)

	target, err := s.CreateWikiPage("", "Target Page", "concept", "", "target body", "")
	if err != nil {
		t.Fatalf("CreateWikiPage target: %v", err)
	}

	// Give a second page a slug identical to the first page's id.
	decoy, err := s.CreateWikiPage(target.ID, "Decoy Page", "concept", "", "decoy body", "")
	if err != nil {
		t.Fatalf("CreateWikiPage decoy: %v", err)
	}
	if decoy.Slug != target.ID {
		t.Fatalf("decoy slug = %q, want %q (the collision this test relies on)", decoy.Slug, target.ID)
	}

	got, err := s.GetWikiPage(target.ID)
	if err != nil {
		t.Fatalf("GetWikiPage: %v", err)
	}
	if got.ID != target.ID {
		t.Errorf("GetWikiPage(%q) resolved to id %q, want the id match (%q) to win over the slug match (%q)",
			target.ID, got.ID, target.ID, decoy.ID)
	}
}

func TestUpdateWikiPageStatus(t *testing.T) {
	s := testStore(t)

	p, _ := s.CreateWikiPage("", "Auth Model", "concept", "", "b", "")

	superseded := "superseded"
	got, err := s.UpdateWikiPage(p.Slug, WikiFields{Status: &superseded})
	if err != nil {
		t.Fatalf("UpdateWikiPage: %v", err)
	}
	if got.Status != "superseded" {
		t.Errorf("Status = %q, want superseded", got.Status)
	}

	bogus := "draft"
	if _, err := s.UpdateWikiPage(p.Slug, WikiFields{Status: &bogus}); !errors.Is(err, ErrInvalid) {
		t.Errorf("err = %v, want ErrInvalid", err)
	}
}

func TestListWikiPagesFilters(t *testing.T) {
	s := testStore(t)

	s.CreateWikiPage("", "One", "concept", "", "a", "")
	spec, _ := s.CreateWikiPage("", "Two", "spec", "", "b", "")

	concepts, _ := s.ListWikiPages("concept", "", "", 0)
	if len(concepts) != 1 || concepts[0].Title != "One" {
		t.Errorf("concepts = %+v", concepts)
	}

	stale := "stale"
	s.UpdateWikiPage(spec.ID, WikiFields{Status: &stale})
	current, _ := s.ListWikiPages("", "current", "", 0)
	if len(current) != 1 || current[0].Title != "One" {
		t.Errorf("current pages = %+v", current)
	}
}

func TestListWikiPagesByProject(t *testing.T) {
	s := testStore(t)
	pid := seedProject(t, s)

	s.CreateWikiPage("", "Scoped", "spec", "", "a", pid)
	s.CreateWikiPage("", "Global", "concept", "", "b", "")

	scoped, err := s.ListWikiPages("", "", pid, 0)
	if err != nil {
		t.Fatalf("ListWikiPages: %v", err)
	}
	if len(scoped) != 1 || scoped[0].Title != "Scoped" {
		t.Errorf("scoped pages = %+v, want just Scoped", scoped)
	}
}

// TestListWikiPagesNullProjectRoundTrips checks that a cross-project page
// (no project_id) round-trips with an empty ProjectID, and that listing
// scoped to a real project does not pick it up via some NULL-matches-empty
// accident.
func TestListWikiPagesNullProjectRoundTrips(t *testing.T) {
	s := testStore(t)
	pid := seedProject(t, s)

	global, err := s.CreateWikiPage("", "Global", "concept", "", "b", "")
	if err != nil {
		t.Fatalf("CreateWikiPage: %v", err)
	}
	if global.ProjectID != "" {
		t.Errorf("ProjectID = %q, want empty for a cross-project page", global.ProjectID)
	}

	fetched, err := s.GetWikiPage(global.ID)
	if err != nil {
		t.Fatalf("GetWikiPage: %v", err)
	}
	if fetched.ProjectID != "" {
		t.Errorf("GetWikiPage ProjectID = %q, want empty", fetched.ProjectID)
	}

	scoped, err := s.ListWikiPages("", "", pid, 0)
	if err != nil {
		t.Fatalf("ListWikiPages: %v", err)
	}
	for _, p := range scoped {
		if p.ID == global.ID {
			t.Errorf("ListWikiPages(project=%q) included cross-project page %q", pid, p.ID)
		}
	}
}
