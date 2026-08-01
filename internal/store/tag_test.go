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

func TestAddTagNormalizesName(t *testing.T) {
	s := testStore(t)
	eid := seedEpic(t, s)
	st, _ := s.CreateStory(eid, "x", "")

	// Add with different cases and spaces
	if err := s.AddTag("  OAuth  ", "story", st.ID); err != nil {
		t.Fatalf("AddTag with mixed case and spaces: %v", err)
	}

	// Verify the stored form is normalized (lowercase, trimmed)
	tags, err := s.TagsFor("story", st.ID)
	if err != nil {
		t.Fatalf("TagsFor: %v", err)
	}
	if len(tags) != 1 || tags[0] != "oauth" {
		t.Errorf("stored tag = %q, want 'oauth' (normalized)", tags[0])
	}

	// Verify adding the same tag in different form is a no-op
	if err := s.AddTag("OAUTH", "story", st.ID); err != nil {
		t.Fatalf("AddTag with uppercase: %v", err)
	}
	tags, _ = s.TagsFor("story", st.ID)
	if len(tags) != 1 {
		t.Errorf("after adding variant, tags = %v, want still 1 entry", tags)
	}
}

func TestRemoveTagValidatesEntityKind(t *testing.T) {
	s := testStore(t)
	eid := seedEpic(t, s)
	st, _ := s.CreateStory(eid, "x", "")

	s.AddTag("test", "story", st.ID)

	// Try to remove with invalid kind (should be ErrInvalid)
	err := s.RemoveTag("test", "invalid", st.ID)
	if !errors.Is(err, ErrInvalid) {
		t.Errorf("RemoveTag with invalid kind: err = %v, want ErrInvalid", err)
	}
}

func TestRemoveTagValidatesEntity(t *testing.T) {
	s := testStore(t)

	// Try to remove a tag from nonexistent entity (should be ErrNotFound)
	err := s.RemoveTag("oauth", "story", "nope99")
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("RemoveTag nonexistent entity: err = %v, want ErrNotFound", err)
	}
}

func TestEmptyTagsFor(t *testing.T) {
	s := testStore(t)
	eid := seedEpic(t, s)
	st, _ := s.CreateStory(eid, "x", "")

	// TagsFor on entity with no tags should return empty slice, not nil
	tags, err := s.TagsFor("story", st.ID)
	if err != nil {
		t.Fatalf("TagsFor: %v", err)
	}
	if tags == nil {
		t.Errorf("TagsFor returned nil, want empty slice")
	}
	if len(tags) != 0 {
		t.Errorf("TagsFor = %v, want empty slice", tags)
	}
}

func TestEmptyListTags(t *testing.T) {
	s := testStore(t)

	// ListTags when no tags exist should return empty slice, not nil
	tags, err := s.ListTags()
	if err != nil {
		t.Fatalf("ListTags: %v", err)
	}
	if tags == nil {
		t.Errorf("ListTags returned nil, want empty slice")
	}
	if len(tags) != 0 {
		t.Errorf("ListTags = %v, want empty slice", tags)
	}
}

func TestEmptyTaggedWith(t *testing.T) {
	s := testStore(t)

	// TaggedWith for nonexistent tag should return empty slice, not nil
	tagged, err := s.TaggedWith("nonexistent")
	if err != nil {
		t.Fatalf("TaggedWith: %v", err)
	}
	if tagged == nil {
		t.Errorf("TaggedWith returned nil, want empty slice")
	}
	if len(tagged) != 0 {
		t.Errorf("TaggedWith = %v, want empty slice", tagged)
	}
}
