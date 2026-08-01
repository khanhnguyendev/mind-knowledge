package store

import (
	"errors"
	"path/filepath"
	"testing"
)

func testStore(t *testing.T) *Store {
	t.Helper()
	s, err := Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	t.Cleanup(func() { s.Close() })
	return s
}

func TestCreateProjectAssignsDefaults(t *testing.T) {
	s := testStore(t)

	p, err := s.CreateProject("my-app", "/Users/ryan/ws/my-app", "")
	if err != nil {
		t.Fatalf("CreateProject: %v", err)
	}
	if len(p.ID) != 6 {
		t.Errorf("ID = %q, want 6 characters", p.ID)
	}
	if p.Status != "active" {
		t.Errorf("Status = %q, want active", p.Status)
	}
	if p.CreatedAt == "" || p.UpdatedAt == "" {
		t.Errorf("timestamps not set: %+v", p)
	}
}

func TestCreateProjectRejectsEmptyName(t *testing.T) {
	s := testStore(t)

	_, err := s.CreateProject("", "/tmp/x", "")
	if !errors.Is(err, ErrInvalid) {
		t.Errorf("err = %v, want ErrInvalid", err)
	}
}

func TestCreateProjectRejectsDuplicateName(t *testing.T) {
	s := testStore(t)

	if _, err := s.CreateProject("my-app", "/tmp/a", ""); err != nil {
		t.Fatalf("first CreateProject: %v", err)
	}
	_, err := s.CreateProject("my-app", "/tmp/b", "")
	if !errors.Is(err, ErrInvalid) {
		t.Errorf("err = %v, want ErrInvalid for duplicate name", err)
	}
}

func TestGetProjectByIDAndName(t *testing.T) {
	s := testStore(t)

	created, err := s.CreateProject("my-app", "/tmp/a", "")
	if err != nil {
		t.Fatalf("CreateProject: %v", err)
	}

	byID, err := s.GetProject(created.ID)
	if err != nil {
		t.Fatalf("GetProject by id: %v", err)
	}
	byName, err := s.GetProject("my-app")
	if err != nil {
		t.Fatalf("GetProject by name: %v", err)
	}
	if byID.ID != created.ID || byName.ID != created.ID {
		t.Errorf("resolved to %q and %q, want %q", byID.ID, byName.ID, created.ID)
	}
}

func TestGetProjectMissing(t *testing.T) {
	s := testStore(t)

	_, err := s.GetProject("nope99")
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("err = %v, want ErrNotFound", err)
	}
}

// TestGetProjectPrefersIDOverCollidingName constructs the case where one
// project's name is literally another project's id. GetProject must resolve
// such a string to the id match, not whichever row a single "id = ? OR
// name = ?" query happens to return first.
func TestGetProjectPrefersIDOverCollidingName(t *testing.T) {
	s := testStore(t)

	alpha, err := s.CreateProject("alpha", "/tmp/a", "")
	if err != nil {
		t.Fatalf("CreateProject alpha: %v", err)
	}
	// beta's name is exactly alpha's id, so the two lookup keys collide.
	if _, err := s.CreateProject(alpha.ID, "/tmp/b", ""); err != nil {
		t.Fatalf("CreateProject with colliding name: %v", err)
	}

	got, err := s.GetProject(alpha.ID)
	if err != nil {
		t.Fatalf("GetProject: %v", err)
	}
	if got.ID != alpha.ID || got.Name != "alpha" {
		t.Errorf("GetProject(%q) = %+v, want the project whose id that is (alpha)",
			alpha.ID, got)
	}
}

func TestListProjectsFiltersByStatus(t *testing.T) {
	s := testStore(t)

	a, _ := s.CreateProject("alpha", "/tmp/a", "")
	if _, err := s.CreateProject("beta", "/tmp/b", ""); err != nil {
		t.Fatalf("CreateProject beta: %v", err)
	}
	archived := "archived"
	if _, err := s.UpdateProject(a.ID, ProjectFields{Status: &archived}); err != nil {
		t.Fatalf("UpdateProject: %v", err)
	}

	all, err := s.ListProjects("", 0)
	if err != nil {
		t.Fatalf("ListProjects all: %v", err)
	}
	if len(all) != 2 {
		t.Errorf("all projects = %d, want 2", len(all))
	}

	active, err := s.ListProjects("active", 0)
	if err != nil {
		t.Fatalf("ListProjects active: %v", err)
	}
	if len(active) != 1 || active[0].Name != "beta" {
		t.Errorf("active projects = %+v, want just beta", active)
	}
}

func TestUpdateProjectRejectsUnknownStatus(t *testing.T) {
	s := testStore(t)

	p, _ := s.CreateProject("my-app", "/tmp/a", "")
	bogus := "retired"
	_, err := s.UpdateProject(p.ID, ProjectFields{Status: &bogus})
	if !errors.Is(err, ErrInvalid) {
		t.Errorf("err = %v, want ErrInvalid", err)
	}
}

func TestDeleteProjectRemovesIt(t *testing.T) {
	s := testStore(t)

	p, _ := s.CreateProject("my-app", "/tmp/a", "")
	if err := s.DeleteProject(p.ID); err != nil {
		t.Fatalf("DeleteProject: %v", err)
	}
	if _, err := s.GetProject(p.ID); !errors.Is(err, ErrNotFound) {
		t.Errorf("after delete, GetProject err = %v, want ErrNotFound", err)
	}
}
