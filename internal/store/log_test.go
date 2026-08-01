package store

import (
	"errors"
	"testing"
)

func TestAddLogEntry(t *testing.T) {
	s := testStore(t)
	pid := seedProject(t, s)

	entry, err := s.AddLog("brainstorm", pid, "a3f2k1", "broke auth work into 5 stories")
	if err != nil {
		t.Fatalf("AddLog: %v", err)
	}
	if entry.ID == 0 {
		t.Error("ID not assigned")
	}
	if entry.Kind != "brainstorm" || entry.Ref != "a3f2k1" {
		t.Errorf("entry = %+v", entry)
	}
	if entry.TS == "" {
		t.Error("TS not set")
	}
}

func TestAddLogRejectsEmptyKindOrSummary(t *testing.T) {
	s := testStore(t)

	if _, err := s.AddLog("", "", "", "something"); !errors.Is(err, ErrInvalid) {
		t.Errorf("empty kind err = %v, want ErrInvalid", err)
	}
	if _, err := s.AddLog("ingest", "", "", ""); !errors.Is(err, ErrInvalid) {
		t.Errorf("empty summary err = %v, want ErrInvalid", err)
	}
}

func TestAddLogRejectsUnknownProject(t *testing.T) {
	s := testStore(t)

	if _, err := s.AddLog("ingest", "nope99", "", "x"); !errors.Is(err, ErrNotFound) {
		t.Errorf("err = %v, want ErrNotFound", err)
	}
}

func TestAddLogAllowsNoProject(t *testing.T) {
	s := testStore(t)

	if _, err := s.AddLog("ingest", "", "", "captured a cross-project source"); err != nil {
		t.Errorf("project-less log entry rejected: %v", err)
	}
}

func TestListLogNewestFirst(t *testing.T) {
	s := testStore(t)

	s.AddLog("ingest", "", "", "first")
	s.AddLog("query", "", "", "second")

	entries, err := s.ListLog("", "", 0)
	if err != nil {
		t.Fatalf("ListLog: %v", err)
	}
	if len(entries) != 2 {
		t.Fatalf("entries = %d, want 2", len(entries))
	}
	if entries[0].Summary != "second" {
		t.Errorf("first entry = %q, want the newest (second)", entries[0].Summary)
	}
}

func TestListLogFiltersAndLimits(t *testing.T) {
	s := testStore(t)
	pid := seedProject(t, s)

	s.AddLog("ingest", pid, "", "a")
	s.AddLog("query", pid, "", "b")
	s.AddLog("ingest", "", "", "c")

	ingests, _ := s.ListLog("ingest", "", 0)
	if len(ingests) != 2 {
		t.Errorf("ingest entries = %d, want 2", len(ingests))
	}

	scoped, _ := s.ListLog("", pid, 0)
	if len(scoped) != 2 {
		t.Errorf("project entries = %d, want 2", len(scoped))
	}

	limited, _ := s.ListLog("", "", 1)
	if len(limited) != 1 {
		t.Errorf("limited entries = %d, want 1", len(limited))
	}
}
