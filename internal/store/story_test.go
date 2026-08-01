package store

import (
	"errors"
	"strings"
	"testing"
)

func seedEpic(t *testing.T, s *Store) string {
	t.Helper()
	pid := seedProject(t, s)
	e, err := s.CreateEpic(pid, "Auth overhaul", "")
	if err != nil {
		t.Fatalf("CreateEpic: %v", err)
	}
	return e.ID
}

func TestCreateStoryDefaults(t *testing.T) {
	s := testStore(t)
	eid := seedEpic(t, s)

	st, err := s.CreateStory(eid, "add login endpoint", "")
	if err != nil {
		t.Fatalf("CreateStory: %v", err)
	}
	if st.Status != "backlog" {
		t.Errorf("Status = %q, want backlog", st.Status)
	}
	if st.Priority != "med" {
		t.Errorf("Priority = %q, want med", st.Priority)
	}
	if st.EpicID != eid {
		t.Errorf("EpicID = %q, want %q", st.EpicID, eid)
	}
}

func TestCreateStoryRejectsUnknownEpic(t *testing.T) {
	s := testStore(t)

	_, err := s.CreateStory("nope99", "x", "")
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("err = %v, want ErrNotFound", err)
	}
}

func TestUpdateStoryValidatesEnums(t *testing.T) {
	s := testStore(t)
	eid := seedEpic(t, s)
	st, _ := s.CreateStory(eid, "x", "")

	bogusStatus := "closed"
	if _, err := s.UpdateStory(st.ID, StoryFields{Status: &bogusStatus}); !errors.Is(err, ErrInvalid) {
		t.Errorf("status err = %v, want ErrInvalid", err)
	}

	bogusPriority := "urgent"
	if _, err := s.UpdateStory(st.ID, StoryFields{Priority: &bogusPriority}); !errors.Is(err, ErrInvalid) {
		t.Errorf("priority err = %v, want ErrInvalid", err)
	}
}

func TestUpdateStoryAcceptsAnyTransition(t *testing.T) {
	s := testStore(t)
	eid := seedEpic(t, s)
	st, _ := s.CreateStory(eid, "x", "")

	// The binary does not enforce workflow order: backlog straight to done
	// with no plan must succeed. Drift is doctor's job, not the store's.
	done := "done"
	updated, err := s.UpdateStory(st.ID, StoryFields{Status: &done})
	if err != nil {
		t.Fatalf("UpdateStory: %v", err)
	}
	if updated.Status != "done" {
		t.Errorf("Status = %q, want done", updated.Status)
	}
}

func TestAppendNotesAccumulates(t *testing.T) {
	s := testStore(t)
	eid := seedEpic(t, s)
	st, _ := s.CreateStory(eid, "x", "")

	first := "ran tests: 12 passed"
	if _, err := s.UpdateStory(st.ID, StoryFields{AppendNotes: &first}); err != nil {
		t.Fatalf("first append: %v", err)
	}
	second := "reviewed: no findings"
	got, err := s.UpdateStory(st.ID, StoryFields{AppendNotes: &second})
	if err != nil {
		t.Fatalf("second append: %v", err)
	}

	if !strings.Contains(got.Notes, first) || !strings.Contains(got.Notes, second) {
		t.Errorf("Notes = %q, want both appends", got.Notes)
	}
	if strings.Index(got.Notes, first) > strings.Index(got.Notes, second) {
		t.Errorf("appends out of order: %q", got.Notes)
	}
}

func TestAppendNotesConflictsWithNotes(t *testing.T) {
	s := testStore(t)
	eid := seedEpic(t, s)
	st, _ := s.CreateStory(eid, "x", "")

	replace := "replaced"
	appendText := "appended"
	_, err := s.UpdateStory(st.ID, StoryFields{Notes: &replace, AppendNotes: &appendText})
	if !errors.Is(err, ErrInvalid) {
		t.Errorf("err = %v, want ErrInvalid when both notes fields are set", err)
	}
}

func TestListStoriesByProject(t *testing.T) {
	s := testStore(t)
	pid := seedProject(t, s)
	e1, _ := s.CreateEpic(pid, "One", "")
	e2, _ := s.CreateEpic(pid, "Two", "")
	s.CreateStory(e1.ID, "a", "")
	s.CreateStory(e2.ID, "b", "")

	got, err := s.ListStories(StoryFilter{ProjectID: pid})
	if err != nil {
		t.Fatalf("ListStories: %v", err)
	}
	if len(got) != 2 {
		t.Errorf("stories = %d, want 2", len(got))
	}
}

func TestListStoriesByStatusAndPriority(t *testing.T) {
	s := testStore(t)
	eid := seedEpic(t, s)
	a, _ := s.CreateStory(eid, "a", "")
	s.CreateStory(eid, "b", "")

	ready := "ready"
	high := "high"
	if _, err := s.UpdateStory(a.ID, StoryFields{Status: &ready, Priority: &high}); err != nil {
		t.Fatalf("UpdateStory: %v", err)
	}

	byStatus, _ := s.ListStories(StoryFilter{EpicID: eid, Status: "ready"})
	if len(byStatus) != 1 || byStatus[0].ID != a.ID {
		t.Errorf("by status = %+v, want just a", byStatus)
	}

	byPriority, _ := s.ListStories(StoryFilter{EpicID: eid, Priority: "high"})
	if len(byPriority) != 1 || byPriority[0].ID != a.ID {
		t.Errorf("by priority = %+v, want just a", byPriority)
	}
}

func TestDeleteEpicCascadesToStories(t *testing.T) {
	s := testStore(t)
	eid := seedEpic(t, s)
	st, _ := s.CreateStory(eid, "x", "")

	if err := s.DeleteEpic(eid); err != nil {
		t.Fatalf("DeleteEpic: %v", err)
	}
	if _, err := s.GetStory(st.ID); !errors.Is(err, ErrNotFound) {
		t.Errorf("story survived epic delete: %v", err)
	}
}

func TestUpdateStoryMoveToEpicRecomputesPosition(t *testing.T) {
	s := testStore(t)
	pid := seedProject(t, s)
	e1, _ := s.CreateEpic(pid, "Epic One", "")
	e2, _ := s.CreateEpic(pid, "Epic Two", "")

	st1, _ := s.CreateStory(e1.ID, "a", "")
	s.CreateStory(e1.ID, "b", "")

	_, _ = s.CreateStory(e2.ID, "existing one", "")
	last, _ := s.CreateStory(e2.ID, "existing two", "")
	oldE2MaxPos := last.Position

	newEpic := e2.ID
	updated, err := s.UpdateStory(st1.ID, StoryFields{EpicID: &newEpic})
	if err != nil {
		t.Fatalf("UpdateStory: %v", err)
	}
	if updated.EpicID != e2.ID {
		t.Errorf("EpicID = %q, want %q", updated.EpicID, e2.ID)
	}
	if updated.Position <= oldE2MaxPos {
		t.Errorf("position = %d, want > %d (should be after existing siblings)",
			updated.Position, oldE2MaxPos)
	}
}

func TestUpdateStoryMoveToEpicWithExplicitPositionDoesNotRecompute(t *testing.T) {
	s := testStore(t)
	pid := seedProject(t, s)
	e1, _ := s.CreateEpic(pid, "Epic One", "")
	e2, _ := s.CreateEpic(pid, "Epic Two", "")

	st1, _ := s.CreateStory(e1.ID, "a", "")
	s.CreateStory(e2.ID, "existing", "")

	newEpic := e2.ID
	explicitPos := 99
	updated, err := s.UpdateStory(st1.ID, StoryFields{EpicID: &newEpic, Position: &explicitPos})
	if err != nil {
		t.Fatalf("UpdateStory: %v", err)
	}
	if updated.EpicID != e2.ID {
		t.Errorf("EpicID = %q, want %q", updated.EpicID, e2.ID)
	}
	if updated.Position != 99 {
		t.Errorf("position = %d, want 99 (should use explicit position)", updated.Position)
	}
}
