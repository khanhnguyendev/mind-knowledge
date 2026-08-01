// Package store owns all SQLite access for mk. Callers never open the
// database directly; they go through Store.
package store

import (
	"database/sql"
	"embed"
	"errors"
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	_ "modernc.org/sqlite"
)

//go:embed migrations/*.sql
var migrationFS embed.FS

// Sentinel errors. The CLI maps these to process exit codes:
// ErrNotFound -> 1, ErrInvalid -> 2, ErrDB -> 3.
var (
	ErrNotFound = errors.New("not found")
	ErrInvalid  = errors.New("invalid input")
	ErrDB       = errors.New("database error")
)

// Now returns the current time. Tests override it for deterministic
// timestamps.
var Now = func() time.Time { return time.Now().UTC() }

// Store is a handle on the mk database.
type Store struct {
	db *sql.DB
}

// DefaultPath returns the database location: $MK_DB when set, otherwise
// ~/.mind-knowledge/mk.db.
func DefaultPath() (string, error) {
	if p := os.Getenv("MK_DB"); p != "" {
		return p, nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("%w: locating home directory: %v", ErrDB, err)
	}
	return filepath.Join(home, ".mind-knowledge", "mk.db"), nil
}

// Open opens the database at path, creating parent directories and
// applying any pending migrations. It is safe to call on an existing
// database.
func Open(path string) (*Store, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, fmt.Errorf("%w: creating %s: %v", ErrDB, filepath.Dir(path), err)
	}

	dsn := path + "?_pragma=busy_timeout(5000)" +
		"&_pragma=journal_mode(WAL)" +
		"&_pragma=foreign_keys(1)"

	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("%w: opening %s: %v", ErrDB, path, err)
	}
	if err := db.Ping(); err != nil {
		db.Close()
		return nil, fmt.Errorf("%w: opening %s: %v", ErrDB, path, err)
	}

	s := &Store{db: db}
	if err := s.migrate(); err != nil {
		db.Close()
		return nil, err
	}
	return s, nil
}

// Close releases the database handle.
func (s *Store) Close() error { return s.db.Close() }

// migrate applies every embedded migration whose version exceeds the
// highest already recorded, in filename order.
func (s *Store) migrate() error {
	_, err := s.db.Exec(
		`CREATE TABLE IF NOT EXISTS schema_migrations (version INTEGER PRIMARY KEY)`)
	if err != nil {
		return fmt.Errorf("%w: creating schema_migrations: %v", ErrDB, err)
	}

	var applied int
	if err := s.db.QueryRow(
		`SELECT COALESCE(MAX(version), 0) FROM schema_migrations`).Scan(&applied); err != nil {
		return fmt.Errorf("%w: reading schema_migrations: %v", ErrDB, err)
	}

	entries, err := migrationFS.ReadDir("migrations")
	if err != nil {
		return fmt.Errorf("%w: reading embedded migrations: %v", ErrDB, err)
	}
	names := make([]string, 0, len(entries))
	for _, e := range entries {
		names = append(names, e.Name())
	}
	sort.Strings(names)

	for _, name := range names {
		version := migrationVersion(name)
		if version <= applied {
			continue
		}
		body, err := migrationFS.ReadFile("migrations/" + name)
		if err != nil {
			return fmt.Errorf("%w: reading migration %s: %v", ErrDB, name, err)
		}
		tx, err := s.db.Begin()
		if err != nil {
			return fmt.Errorf("%w: starting migration %s: %v", ErrDB, name, err)
		}
		if _, err := tx.Exec(string(body)); err != nil {
			tx.Rollback()
			return fmt.Errorf("%w: applying migration %s: %v", ErrDB, name, err)
		}
		if _, err := tx.Exec(
			`INSERT INTO schema_migrations (version) VALUES (?)`, version); err != nil {
			tx.Rollback()
			return fmt.Errorf("%w: recording migration %s: %v", ErrDB, name, err)
		}
		if err := tx.Commit(); err != nil {
			return fmt.Errorf("%w: committing migration %s: %v", ErrDB, name, err)
		}
	}
	return nil
}

// migrationVersion extracts the leading integer from a filename such as
// "0002_fts.sql". Unparseable names yield 0 and are therefore skipped.
func migrationVersion(name string) int {
	digits := name
	if i := strings.IndexByte(name, '_'); i >= 0 {
		digits = name[:i]
	}
	version := 0
	for _, c := range digits {
		if c < '0' || c > '9' {
			return 0
		}
		version = version*10 + int(c-'0')
	}
	return version
}

const base36 = "0123456789abcdefghijklmnopqrstuvwxyz"

// NewID returns a random 6-character base36 identifier.
func NewID() string {
	b := make([]byte, 6)
	for i := range b {
		b[i] = base36[rand.Intn(len(base36))]
	}
	return string(b)
}

// uniqueID returns an id not already present in the given table. It gives
// up after a bounded number of attempts rather than looping forever.
func (s *Store) uniqueID(table string) (string, error) {
	for attempt := 0; attempt < 20; attempt++ {
		id := NewID()
		var found string
		err := s.db.QueryRow(
			fmt.Sprintf(`SELECT id FROM %s WHERE id = ?`, table), id).Scan(&found)
		if errors.Is(err, sql.ErrNoRows) {
			return id, nil
		}
		if err != nil {
			return "", fmt.Errorf("%w: checking id in %s: %v", ErrDB, table, err)
		}
	}
	return "", fmt.Errorf("%w: could not find a free id in %s", ErrDB, table)
}

// timestamp returns the current time formatted for storage.
func timestamp() string { return Now().Format(time.RFC3339) }
