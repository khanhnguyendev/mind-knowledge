package store

import "testing"

// countLinks reports how many edges name (kind, id) on either end.
func countLinks(t *testing.T, s *Store, kind, id string) int {
	t.Helper()
	var n int
	err := s.db.QueryRow(
		`SELECT COUNT(*) FROM links
		 WHERE (from_kind = ? AND from_id = ?) OR (to_kind = ? AND to_id = ?)`,
		kind, id, kind, id).Scan(&n)
	if err != nil {
		t.Fatalf("counting links: %v", err)
	}
	return n
}

// countEntityTags reports how many entity_tags rows name (kind, id).
func countEntityTags(t *testing.T, s *Store, kind, id string) int {
	t.Helper()
	var n int
	err := s.db.QueryRow(
		`SELECT COUNT(*) FROM entity_tags WHERE entity_kind = ? AND entity_id = ?`,
		kind, id).Scan(&n)
	if err != nil {
		t.Fatalf("counting entity_tags: %v", err)
	}
	return n
}

func TestDeleteWikiPageRemovesItsLinks(t *testing.T) {
	s := testStore(t)

	src, _ := s.CreateSource("", "An article", "article", "text", "")
	page, _ := s.CreateWikiPage("", "Derived", "summary", "", "b", "")
	other, _ := s.CreateWikiPage("", "Other", "concept", "", "b", "")

	// An outbound edge (page -> source) and an inbound one (other -> page).
	s.AddLink("wiki", page.ID, "source", src.ID, "derived-from")
	s.AddLink("wiki", other.ID, "wiki", page.ID, "references")

	if err := s.DeleteWikiPage(page.ID); err != nil {
		t.Fatalf("DeleteWikiPage: %v", err)
	}

	if n := countLinks(t, s, "wiki", page.ID); n != 0 {
		t.Errorf("%d links still name the deleted page, want 0", n)
	}
}

func TestDeleteWikiPageRemovesItsTags(t *testing.T) {
	s := testStore(t)

	page, _ := s.CreateWikiPage("", "Tagged", "concept", "", "b", "")
	if err := s.AddTag("hot", "wiki", page.ID); err != nil {
		t.Fatalf("AddTag: %v", err)
	}

	if err := s.DeleteWikiPage(page.ID); err != nil {
		t.Fatalf("DeleteWikiPage: %v", err)
	}

	if n := countEntityTags(t, s, "wiki", page.ID); n != 0 {
		t.Errorf("%d entity_tags rows still name the deleted page, want 0", n)
	}
	if tagged, _ := s.TaggedWith("hot"); len(tagged) != 0 {
		t.Errorf("TaggedWith(hot) = %+v, want no phantom entries", tagged)
	}
}

func TestDeleteSourceRemovesItsLinksAndTags(t *testing.T) {
	s := testStore(t)

	src, _ := s.CreateSource("", "An article", "article", "text", "")
	page, _ := s.CreateWikiPage("", "Derived", "summary", "", "b", "")
	s.AddLink("wiki", page.ID, "source", src.ID, "derived-from")
	s.AddTag("hot", "source", src.ID)

	if err := s.DeleteSource(src.ID); err != nil {
		t.Fatalf("DeleteSource: %v", err)
	}

	if n := countLinks(t, s, "source", src.ID); n != 0 {
		t.Errorf("%d links still name the deleted source, want 0", n)
	}
	if n := countEntityTags(t, s, "source", src.ID); n != 0 {
		t.Errorf("%d entity_tags rows still name the deleted source, want 0", n)
	}
}

func TestDeleteStoryRemovesItsLinksAndTags(t *testing.T) {
	s := testStore(t)

	p, _ := s.CreateProject("my-app", "/tmp/my-app", "")
	e, _ := s.CreateEpic(p.ID, "Auth", "")
	st, _ := s.CreateStory(e.ID, "login", "")
	page, _ := s.CreateWikiPage("", "Auth Model", "concept", "", "b", "")

	s.AddLink("story", st.ID, "wiki", page.ID, "implements")
	s.AddTag("hot", "story", st.ID)

	if err := s.DeleteStory(st.ID); err != nil {
		t.Fatalf("DeleteStory: %v", err)
	}

	if n := countLinks(t, s, "story", st.ID); n != 0 {
		t.Errorf("%d links still name the deleted story, want 0", n)
	}
	if n := countEntityTags(t, s, "story", st.ID); n != 0 {
		t.Errorf("%d entity_tags rows still name the deleted story, want 0", n)
	}
	if tagged, _ := s.TaggedWith("hot"); len(tagged) != 0 {
		t.Errorf("TaggedWith(hot) = %+v, want no phantom entries", tagged)
	}
}

func TestDeleteEpicRemovesCascadedStoryReferences(t *testing.T) {
	s := testStore(t)

	p, _ := s.CreateProject("my-app", "/tmp/my-app", "")
	e, _ := s.CreateEpic(p.ID, "Auth", "")
	st, _ := s.CreateStory(e.ID, "login", "")
	page, _ := s.CreateWikiPage("", "Auth Model", "concept", "", "b", "")

	s.AddLink("epic", e.ID, "wiki", page.ID, "references")
	s.AddLink("story", st.ID, "wiki", page.ID, "implements")
	s.AddTag("hot", "epic", e.ID)
	s.AddTag("hot", "story", st.ID)

	if err := s.DeleteEpic(e.ID); err != nil {
		t.Fatalf("DeleteEpic: %v", err)
	}

	// The database cascades epics -> stories, so the story's own edges leak
	// one level down unless DeleteEpic cleans them too.
	if n := countLinks(t, s, "epic", e.ID); n != 0 {
		t.Errorf("%d links still name the deleted epic, want 0", n)
	}
	if n := countLinks(t, s, "story", st.ID); n != 0 {
		t.Errorf("%d links still name the cascaded story, want 0", n)
	}
	if n := countEntityTags(t, s, "epic", e.ID); n != 0 {
		t.Errorf("%d entity_tags rows still name the deleted epic, want 0", n)
	}
	if n := countEntityTags(t, s, "story", st.ID); n != 0 {
		t.Errorf("%d entity_tags rows still name the cascaded story, want 0", n)
	}
}

func TestDeleteProjectRemovesCascadedEpicAndStoryReferences(t *testing.T) {
	s := testStore(t)

	p, _ := s.CreateProject("my-app", "/tmp/my-app", "")
	e, _ := s.CreateEpic(p.ID, "Auth", "")
	st, _ := s.CreateStory(e.ID, "login", "")
	page, _ := s.CreateWikiPage("", "Auth Model", "concept", "", "b", "")

	s.AddLink("project", p.ID, "wiki", page.ID, "references")
	s.AddLink("epic", e.ID, "wiki", page.ID, "references")
	s.AddLink("story", st.ID, "wiki", page.ID, "implements")
	s.AddTag("hot", "project", p.ID)
	s.AddTag("hot", "epic", e.ID)
	s.AddTag("hot", "story", st.ID)

	if err := s.DeleteProject(p.ID); err != nil {
		t.Fatalf("DeleteProject: %v", err)
	}

	for _, ref := range []struct{ kind, id string }{
		{"project", p.ID}, {"epic", e.ID}, {"story", st.ID},
	} {
		if n := countLinks(t, s, ref.kind, ref.id); n != 0 {
			t.Errorf("%d links still name the deleted %s, want 0", n, ref.kind)
		}
		if n := countEntityTags(t, s, ref.kind, ref.id); n != 0 {
			t.Errorf("%d entity_tags rows still name the deleted %s, want 0", n, ref.kind)
		}
	}
	if tagged, _ := s.TaggedWith("hot"); len(tagged) != 0 {
		t.Errorf("TaggedWith(hot) = %+v, want no phantom entries", tagged)
	}
}

func TestDeleteLeavesSurvivingEntitiesEdgesAlone(t *testing.T) {
	s := testStore(t)

	src, _ := s.CreateSource("", "An article", "article", "text", "")
	doomed, _ := s.CreateWikiPage("", "Doomed", "summary", "", "b", "")
	keeper, _ := s.CreateWikiPage("", "Keeper", "summary", "", "b", "")

	s.AddLink("wiki", doomed.ID, "source", src.ID, "derived-from")
	s.AddLink("wiki", keeper.ID, "source", src.ID, "derived-from")
	s.AddTag("hot", "wiki", keeper.ID)

	if err := s.DeleteWikiPage(doomed.ID); err != nil {
		t.Fatalf("DeleteWikiPage: %v", err)
	}

	if n := countLinks(t, s, "wiki", keeper.ID); n != 1 {
		t.Errorf("surviving page has %d links, want 1", n)
	}
	if n := countEntityTags(t, s, "wiki", keeper.ID); n != 1 {
		t.Errorf("surviving page has %d tags, want 1", n)
	}
}

func TestDanglingLinksReportsEdgesWithMissingEndpoints(t *testing.T) {
	s := testStore(t)

	src, _ := s.CreateSource("", "An article", "article", "text", "")
	page, _ := s.CreateWikiPage("", "Derived", "summary", "", "b", "")
	s.AddLink("wiki", page.ID, "source", src.ID, "derived-from")

	if got, err := s.DanglingLinks(); err != nil || len(got) != 0 {
		t.Fatalf("DanglingLinks() = %+v, %v, want no findings on a healthy graph", got, err)
	}

	// Forge the leak the old Delete* paths used to produce.
	if _, err := s.db.Exec(`DELETE FROM wiki_pages WHERE id = ?`, page.ID); err != nil {
		t.Fatalf("forcing a dangling edge: %v", err)
	}

	got, err := s.DanglingLinks()
	if err != nil {
		t.Fatalf("DanglingLinks: %v", err)
	}
	if len(got) != 1 || got[0].FromID != page.ID {
		t.Errorf("DanglingLinks() = %+v, want the edge from the deleted page", got)
	}
}
