package store

import (
	"errors"
	"testing"
)

func seedProject(t *testing.T, s *Store) string {
	t.Helper()
	p, err := s.CreateProject("my-app", "/tmp/my-app", "")
	if err != nil {
		t.Fatalf("CreateProject: %v", err)
	}
	return p.ID
}

func TestCreateEpicDefaults(t *testing.T) {
	s := testStore(t)
	pid := seedProject(t, s)

	e, err := s.CreateEpic(pid, "Auth overhaul", "")
	if err != nil {
		t.Fatalf("CreateEpic: %v", err)
	}
	if e.Status != "backlog" {
		t.Errorf("Status = %q, want backlog", e.Status)
	}
	if e.ProjectID != pid {
		t.Errorf("ProjectID = %q, want %q", e.ProjectID, pid)
	}
}

func TestCreateEpicRejectsUnknownProject(t *testing.T) {
	s := testStore(t)

	_, err := s.CreateEpic("nope99", "Auth overhaul", "")
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("err = %v, want ErrNotFound", err)
	}
}

func TestCreateEpicRejectsEmptyTitle(t *testing.T) {
	s := testStore(t)
	pid := seedProject(t, s)

	_, err := s.CreateEpic(pid, "", "")
	if !errors.Is(err, ErrInvalid) {
		t.Errorf("err = %v, want ErrInvalid", err)
	}
}

func TestCreateEpicAssignsIncreasingPositions(t *testing.T) {
	s := testStore(t)
	pid := seedProject(t, s)

	first, _ := s.CreateEpic(pid, "One", "")
	second, _ := s.CreateEpic(pid, "Two", "")
	if second.Position <= first.Position {
		t.Errorf("positions %d then %d, want increasing", first.Position, second.Position)
	}
}

func TestListEpicsFilters(t *testing.T) {
	s := testStore(t)
	pid := seedProject(t, s)

	a, _ := s.CreateEpic(pid, "One", "")
	s.CreateEpic(pid, "Two", "")
	done := "done"
	if _, err := s.UpdateEpic(a.ID, EpicFields{Status: &done}); err != nil {
		t.Fatalf("UpdateEpic: %v", err)
	}

	all, _ := s.ListEpics(pid, "", 0)
	if len(all) != 2 {
		t.Errorf("all epics = %d, want 2", len(all))
	}

	open, _ := s.ListEpics(pid, "backlog", 0)
	if len(open) != 1 || open[0].Title != "Two" {
		t.Errorf("backlog epics = %+v, want just Two", open)
	}
}

func TestUpdateEpicRejectsUnknownStatus(t *testing.T) {
	s := testStore(t)
	pid := seedProject(t, s)
	e, _ := s.CreateEpic(pid, "One", "")

	bogus := "ready"
	_, err := s.UpdateEpic(e.ID, EpicFields{Status: &bogus})
	if !errors.Is(err, ErrInvalid) {
		t.Errorf("err = %v, want ErrInvalid (ready is a story status, not an epic status)", err)
	}
}

func TestDeleteProjectCascadesToEpics(t *testing.T) {
	s := testStore(t)
	pid := seedProject(t, s)
	e, _ := s.CreateEpic(pid, "One", "")

	if err := s.DeleteProject(pid); err != nil {
		t.Fatalf("DeleteProject: %v", err)
	}
	if _, err := s.GetEpic(e.ID); !errors.Is(err, ErrNotFound) {
		t.Errorf("epic survived project delete: %v", err)
	}
}

func TestUpdateEpicMoveToProjectRecomputesPosition(t *testing.T) {
	s := testStore(t)
	p1, _ := s.CreateProject("proj1", "/tmp/proj1", "")
	p2, _ := s.CreateProject("proj2", "/tmp/proj2", "")

	// Create epics in proj1
	e1, _ := s.CreateEpic(p1.ID, "Epic One", "")
	e2, _ := s.CreateEpic(p1.ID, "Epic Two", "")

	// Create existing epics in proj2
	_, _ = s.CreateEpic(p2.ID, "Existing One", "")
	e4, _ := s.CreateEpic(p2.ID, "Existing Two", "")
	oldP2MaxPos := e4.Position

	// Move e1 to proj2 without specifying position
	newProj := p2.ID
	updated, err := s.UpdateEpic(e1.ID, EpicFields{ProjectID: &newProj})
	if err != nil {
		t.Fatalf("UpdateEpic: %v", err)
	}

	// Verify it moved to the right project
	if updated.ProjectID != p2.ID {
		t.Errorf("ProjectID = %q, want %q", updated.ProjectID, p2.ID)
	}

	// Verify position was recomputed (should be after max position in proj2)
	if updated.Position <= oldP2MaxPos {
		t.Errorf("position = %d, want > %d (position should be after existing siblings)", updated.Position, oldP2MaxPos)
	}

	// Verify it still exists in proj2 and has the right position
	retrieved, err := s.GetEpic(e1.ID)
	if err != nil {
		t.Fatalf("GetEpic: %v", err)
	}
	if retrieved.ProjectID != p2.ID || retrieved.Position != updated.Position {
		t.Errorf("retrieved epic = %+v, want ProjectID %q and Position %d", retrieved, p2.ID, updated.Position)
	}

	// Verify proj1 still has e2
	e2Check, _ := s.GetEpic(e2.ID)
	if e2Check.ProjectID != p1.ID {
		t.Errorf("e2 should still be in proj1")
	}
}

func TestUpdateEpicMoveToProjectWithExplicitPositionDoesNotRecompute(t *testing.T) {
	s := testStore(t)
	p1, _ := s.CreateProject("proj1", "/tmp/proj1", "")
	p2, _ := s.CreateProject("proj2", "/tmp/proj2", "")

	// Create epics in proj1
	e1, _ := s.CreateEpic(p1.ID, "Epic One", "")

	// Create existing epics in proj2 to set up state
	_, _ = s.CreateEpic(p2.ID, "Existing One", "")
	_, _ = s.CreateEpic(p2.ID, "Existing Two", "")

	// Move e1 to proj2 WITH explicit position 99
	newProj := p2.ID
	explicitPos := 99
	updated, err := s.UpdateEpic(e1.ID, EpicFields{ProjectID: &newProj, Position: &explicitPos})
	if err != nil {
		t.Fatalf("UpdateEpic: %v", err)
	}

	// Verify it moved to the right project
	if updated.ProjectID != p2.ID {
		t.Errorf("ProjectID = %q, want %q", updated.ProjectID, p2.ID)
	}

	// Verify position is exactly what was specified (not recomputed)
	if updated.Position != 99 {
		t.Errorf("position = %d, want 99 (should use explicit position)", updated.Position)
	}
}
