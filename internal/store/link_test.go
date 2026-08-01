package store

import (
	"errors"
	"testing"
)

func TestAddLinkResolvesEndpoints(t *testing.T) {
	s := testStore(t)

	src, _ := s.CreateSource("", "An article", "article", "body", "")
	page, _ := s.CreateWikiPage("", "Auth Model", "summary", "", "b", "")

	// The wiki page is addressed by slug; the link must store its id.
	link, err := s.AddLink("wiki", page.Slug, "source", src.ID, "derived-from")
	if err != nil {
		t.Fatalf("AddLink: %v", err)
	}
	if link.FromID != page.ID {
		t.Errorf("FromID = %q, want the page id %q", link.FromID, page.ID)
	}
	if link.ToID != src.ID {
		t.Errorf("ToID = %q, want %q", link.ToID, src.ID)
	}
}

func TestAddLinkRejectsUnknownRelation(t *testing.T) {
	s := testStore(t)
	page, _ := s.CreateWikiPage("", "A", "concept", "", "b", "")

	_, err := s.AddLink("wiki", page.ID, "wiki", page.ID, "cites")
	if !errors.Is(err, ErrInvalid) {
		t.Errorf("err = %v, want ErrInvalid", err)
	}
}

func TestAddLinkRejectsUnknownKind(t *testing.T) {
	s := testStore(t)
	page, _ := s.CreateWikiPage("", "A", "concept", "", "b", "")

	_, err := s.AddLink("page", page.ID, "wiki", page.ID, "references")
	if !errors.Is(err, ErrInvalid) {
		t.Errorf("err = %v, want ErrInvalid", err)
	}
}

func TestAddLinkRejectsMissingEndpoint(t *testing.T) {
	s := testStore(t)
	page, _ := s.CreateWikiPage("", "A", "concept", "", "b", "")

	_, err := s.AddLink("wiki", page.ID, "source", "nope99", "derived-from")
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("err = %v, want ErrNotFound", err)
	}
}

func TestAddLinkIsIdempotent(t *testing.T) {
	s := testStore(t)
	a, _ := s.CreateWikiPage("", "A", "concept", "", "b", "")
	b, _ := s.CreateWikiPage("", "B", "concept", "", "b", "")

	if _, err := s.AddLink("wiki", a.ID, "wiki", b.ID, "references"); err != nil {
		t.Fatalf("first AddLink: %v", err)
	}
	if _, err := s.AddLink("wiki", a.ID, "wiki", b.ID, "references"); err != nil {
		t.Fatalf("repeat AddLink should be a no-op, got: %v", err)
	}

	links, _ := s.ListLinks("wiki", a.ID, "", "", "")
	if len(links) != 1 {
		t.Errorf("links = %d, want 1 after adding the same edge twice", len(links))
	}
}

func TestListLinksFiltersInBothDirections(t *testing.T) {
	s := testStore(t)
	a, _ := s.CreateWikiPage("", "A", "concept", "", "b", "")
	b, _ := s.CreateWikiPage("", "B", "concept", "", "b", "")
	c, _ := s.CreateWikiPage("", "C", "concept", "", "b", "")

	s.AddLink("wiki", a.ID, "wiki", b.ID, "references")
	s.AddLink("wiki", c.ID, "wiki", b.ID, "supersedes")

	outbound, _ := s.ListLinks("wiki", a.ID, "", "", "")
	if len(outbound) != 1 {
		t.Errorf("outbound from a = %d, want 1", len(outbound))
	}

	inbound, _ := s.ListLinks("", "", "wiki", b.ID, "")
	if len(inbound) != 2 {
		t.Errorf("inbound to b = %d, want 2", len(inbound))
	}

	byRelation, _ := s.ListLinks("", "", "", "", "supersedes")
	if len(byRelation) != 1 {
		t.Errorf("supersedes links = %d, want 1", len(byRelation))
	}
}

func TestRemoveLink(t *testing.T) {
	s := testStore(t)
	a, _ := s.CreateWikiPage("", "A", "concept", "", "b", "")
	b, _ := s.CreateWikiPage("", "B", "concept", "", "b", "")

	s.AddLink("wiki", a.ID, "wiki", b.ID, "references")
	if err := s.RemoveLink("wiki", a.ID, "wiki", b.ID, "references"); err != nil {
		t.Fatalf("RemoveLink: %v", err)
	}

	links, _ := s.ListLinks("wiki", a.ID, "", "", "")
	if len(links) != 0 {
		t.Errorf("links = %d after removal, want 0", len(links))
	}

	if err := s.RemoveLink("wiki", a.ID, "wiki", b.ID, "references"); !errors.Is(err, ErrNotFound) {
		t.Errorf("removing a missing link err = %v, want ErrNotFound", err)
	}
}

func TestRemoveLinkRejectsUnknownRelation(t *testing.T) {
	s := testStore(t)
	a, _ := s.CreateWikiPage("", "A", "concept", "", "b", "")
	b, _ := s.CreateWikiPage("", "B", "concept", "", "b", "")

	err := s.RemoveLink("wiki", a.ID, "wiki", b.ID, "cites")
	if !errors.Is(err, ErrInvalid) {
		t.Errorf("err = %v, want ErrInvalid (not ErrNotFound) for a bad relation", err)
	}
}

func TestListLinksRejectsUnknownRelation(t *testing.T) {
	s := testStore(t)

	_, err := s.ListLinks("", "", "", "", "cites")
	if !errors.Is(err, ErrInvalid) {
		t.Errorf("err = %v, want ErrInvalid", err)
	}
}
