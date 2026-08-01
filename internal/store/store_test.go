package store

import (
	"path/filepath"
	"testing"
)

func TestOpenCreatesSchema(t *testing.T) {
	path := filepath.Join(t.TempDir(), "test.db")

	s, err := Open(path)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer s.Close()

	tables := []string{
		"projects", "epics", "stories", "sources",
		"wiki_pages", "links", "tags", "entity_tags", "log",
	}
	for _, name := range tables {
		var got string
		err := s.db.QueryRow(
			`SELECT name FROM sqlite_master WHERE type='table' AND name=?`, name,
		).Scan(&got)
		if err != nil {
			t.Errorf("table %q missing: %v", name, err)
		}
	}
}

func TestOpenIsIdempotent(t *testing.T) {
	path := filepath.Join(t.TempDir(), "test.db")

	s1, err := Open(path)
	if err != nil {
		t.Fatalf("first Open: %v", err)
	}
	s1.Close()

	s2, err := Open(path)
	if err != nil {
		t.Fatalf("second Open: %v", err)
	}
	defer s2.Close()

	var version int
	if err := s2.db.QueryRow(`SELECT MAX(version) FROM schema_migrations`).Scan(&version); err != nil {
		t.Fatalf("read schema_migrations: %v", err)
	}
	if version != 2 {
		t.Errorf("schema version = %d, want 2", version)
	}
}

func TestNewIDShape(t *testing.T) {
	seen := map[string]bool{}
	for i := 0; i < 1000; i++ {
		id := NewID()
		if len(id) != 6 {
			t.Fatalf("NewID() = %q, want length 6", id)
		}
		for _, c := range id {
			isDigit := c >= '0' && c <= '9'
			isLower := c >= 'a' && c <= 'z'
			if !isDigit && !isLower {
				t.Fatalf("NewID() = %q contains non-base36 rune %q", id, c)
			}
		}
		seen[id] = true
	}
	if len(seen) < 990 {
		t.Errorf("only %d unique ids in 1000 draws, want >= 990", len(seen))
	}
}

func TestDefaultPathHonorsEnv(t *testing.T) {
	t.Setenv("MK_DB", "/tmp/custom.db")
	got, err := DefaultPath()
	if err != nil {
		t.Fatalf("DefaultPath: %v", err)
	}
	if got != "/tmp/custom.db" {
		t.Errorf("DefaultPath() = %q, want /tmp/custom.db", got)
	}
}
