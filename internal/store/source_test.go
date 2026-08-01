package store

import (
	"errors"
	"testing"
)

func TestCreateSourceHashesBody(t *testing.T) {
	s := testStore(t)

	src, err := s.CreateSource("https://example.com/a", "An article", "article", "hello", "")
	if err != nil {
		t.Fatalf("CreateSource: %v", err)
	}
	// SHA-256 of "hello".
	const want = "2cf24dba5fb0a30e26e83b2ac5b9e29e1b161e5c1fa7425e73043362938b9824"
	if src.ContentHash != want {
		t.Errorf("ContentHash = %q, want %q", src.ContentHash, want)
	}
	if src.IngestedAt == "" {
		t.Error("IngestedAt not set")
	}
}

func TestCreateSourceRejectsEmptyTitle(t *testing.T) {
	s := testStore(t)

	_, err := s.CreateSource("", "", "note", "body", "")
	if !errors.Is(err, ErrInvalid) {
		t.Errorf("err = %v, want ErrInvalid", err)
	}
}

func TestCreateSourceRejectsUnknownKind(t *testing.T) {
	s := testStore(t)

	_, err := s.CreateSource("", "A title", "video", "body", "")
	if !errors.Is(err, ErrInvalid) {
		t.Errorf("err = %v, want ErrInvalid", err)
	}
}

func TestCreateSourceRequiresBodyOrAsset(t *testing.T) {
	s := testStore(t)

	_, err := s.CreateSource("", "A title", "note", "", "")
	if !errors.Is(err, ErrInvalid) {
		t.Errorf("err = %v, want ErrInvalid when both body and asset path are empty", err)
	}

	if _, err := s.CreateSource("", "A diagram", "asset", "", "/tmp/x.png"); err != nil {
		t.Errorf("asset-only source rejected: %v", err)
	}
}

func TestFindSourceByHashDetectsDuplicate(t *testing.T) {
	s := testStore(t)

	first, err := s.CreateSource("", "A title", "note", "same body", "")
	if err != nil {
		t.Fatalf("CreateSource: %v", err)
	}

	found, err := s.FindSourceByHash(first.ContentHash)
	if err != nil {
		t.Fatalf("FindSourceByHash: %v", err)
	}
	if found.ID != first.ID {
		t.Errorf("found %q, want %q", found.ID, first.ID)
	}

	if _, err := s.FindSourceByHash("deadbeef"); !errors.Is(err, ErrNotFound) {
		t.Errorf("unknown hash err = %v, want ErrNotFound", err)
	}
}

func TestListSourcesFiltersByKind(t *testing.T) {
	s := testStore(t)

	s.CreateSource("", "One", "article", "a", "")
	s.CreateSource("", "Two", "paper", "b", "")

	all, _ := s.ListSources("", 0)
	if len(all) != 2 {
		t.Errorf("all sources = %d, want 2", len(all))
	}

	papers, _ := s.ListSources("paper", 0)
	if len(papers) != 1 || papers[0].Title != "Two" {
		t.Errorf("papers = %+v, want just Two", papers)
	}
}

func TestDeleteSource(t *testing.T) {
	s := testStore(t)

	src, _ := s.CreateSource("", "One", "note", "a", "")
	if err := s.DeleteSource(src.ID); err != nil {
		t.Fatalf("DeleteSource: %v", err)
	}
	if _, err := s.GetSource(src.ID); !errors.Is(err, ErrNotFound) {
		t.Errorf("err = %v, want ErrNotFound", err)
	}
}
