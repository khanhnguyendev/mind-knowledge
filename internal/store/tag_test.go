package store

import (
	"errors"
	"testing"
)

func TestAddTagAttachesToEntity(t *testing.T) {
	s := testStore(t)
	eid := seedEpic(t, s)
	st, _ := s.CreateStory(eid, "add login endpoint", "")

	if err := s.AddTag("oauth", "story", st.ID); err != nil {
		t.Fatalf("AddTag: %v", err)
	}

	tags, err := s.TagsFor("story", st.ID)
	if err != nil {
		t.Fatalf("TagsFor: %v", err)
	}
	if len(tags) != 1 || tags[0] != "oauth" {
		t.Errorf("tags = %v, want [oauth]", tags)
	}
}

func TestAddTagIsIdempotent(t *testing.T) {
	s := testStore(t)
	eid := seedEpic(t, s)
	st, _ := s.CreateStory(eid, "x", "")

	if err := s.AddTag("oauth", "story", st.ID); err != nil {
		t.Fatalf("first AddTag: %v", err)
	}
	if err := s.AddTag("oauth", "story", st.ID); err != nil {
		t.Fatalf("repeat AddTag should be a no-op, got: %v", err)
	}

	tags, _ := s.TagsFor("story", st.ID)
	if len(tags) != 1 {
		t.Errorf("tags = %v, want one entry", tags)
	}
}

func TestAddTagRejectsEmptyName(t *testing.T) {
	s := testStore(t)
	eid := seedEpic(t, s)
	st, _ := s.CreateStory(eid, "x", "")

	if err := s.AddTag("", "story", st.ID); !errors.Is(err, ErrInvalid) {
		t.Errorf("err = %v, want ErrInvalid", err)
	}
}

func TestAddTagRejectsMissingEntity(t *testing.T) {
	s := testStore(t)

	if err := s.AddTag("oauth", "story", "nope99"); !errors.Is(err, ErrNotFound) {
		t.Errorf("err = %v, want ErrNotFound", err)
	}
}

func TestTaggedWithSpansKinds(t *testing.T) {
	s := testStore(t)
	pid := seedProject(t, s)
	e, _ := s.CreateEpic(pid, "Auth overhaul", "")
	st, _ := s.CreateStory(e.ID, "x", "")

	s.AddTag("oauth", "epic", e.ID)
	s.AddTag("oauth", "story", st.ID)

	tagged, err := s.TaggedWith("oauth")
	if err != nil {
		t.Fatalf("TaggedWith: %v", err)
	}
	if len(tagged) != 2 {
		t.Fatalf("tagged = %+v, want 2 entries", tagged)
	}

	kinds := map[string]bool{}
	for _, item := range tagged {
		kinds[item.FromKind] = true
	}
	if !kinds["epic"] || !kinds["story"] {
		t.Errorf("tagged kinds = %v, want both epic and story", kinds)
	}
}

func TestListTagsSorted(t *testing.T) {
	s := testStore(t)
	eid := seedEpic(t, s)
	st, _ := s.CreateStory(eid, "x", "")

	s.AddTag("zeta", "story", st.ID)
	s.AddTag("alpha", "story", st.ID)

	names, err := s.ListTags()
	if err != nil {
		t.Fatalf("ListTags: %v", err)
	}
	if len(names) != 2 || names[0] != "alpha" || names[1] != "zeta" {
		t.Errorf("tags = %v, want [alpha zeta]", names)
	}
}

func TestRemoveTag(t *testing.T) {
	s := testStore(t)
	eid := seedEpic(t, s)
	st, _ := s.CreateStory(eid, "x", "")

	s.AddTag("oauth", "story", st.ID)
	if err := s.RemoveTag("oauth", "story", st.ID); err != nil {
		t.Fatalf("RemoveTag: %v", err)
	}

	tags, _ := s.TagsFor("story", st.ID)
	if len(tags) != 0 {
		t.Errorf("tags = %v after removal, want none", tags)
	}

	if err := s.RemoveTag("oauth", "story", st.ID); !errors.Is(err, ErrNotFound) {
		t.Errorf("removing an absent tag err = %v, want ErrNotFound", err)
	}
}
