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

func seedEpic(t *testing.T, s *Store, pid string) string {
	t.Helper()
	e, err := s.CreateEpic(pid, "Test Epic", "")
	if err != nil {
		t.Fatalf("CreateEpic: %v", err)
	}
	return e.ID
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
