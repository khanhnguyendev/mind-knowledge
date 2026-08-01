// Package store owns all SQLite access for mk. Callers never open the
// database directly; they go through Store.
package store

import (
	"context"
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

	"modernc.org/sqlite"
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

// Driver error codes we classify. They are spelled out here rather than
// imported from modernc.org/sqlite/lib so this package's dependency
// surface stays at the two packages the project allows.
const (
	sqliteBusy                 = 5    // SQLITE_BUSY
	sqliteLocked               = 6    // SQLITE_LOCKED
	sqliteConstraintUnique     = 2067 // SQLITE_CONSTRAINT_UNIQUE
	sqliteConstraintPrimaryKey = 1555 // SQLITE_CONSTRAINT_PRIMARYKEY
)

// isUniqueViolation reports whether err came from a UNIQUE or PRIMARY KEY
// constraint failure at the database layer.
func isUniqueViolation(err error) bool {
	var sqliteErr *sqlite.Error
	if !errors.As(err, &sqliteErr) {
		return false
	}
	switch sqliteErr.Code() {
	case sqliteConstraintUnique, sqliteConstraintPrimaryKey:
		return true
	default:
		return false
	}
}

// isBusy reports whether err means "another connection holds the lock I
// need", which is a wait-and-retry condition rather than a real failure.
// SQLite reports it with extended codes too (SQLITE_BUSY_SNAPSHOT and
// friends), which carry the primary code in their low byte.
func isBusy(err error) bool {
	var sqliteErr *sqlite.Error
	if !errors.As(err, &sqliteErr) {
		return false
	}
	switch sqliteErr.Code() & 0xff {
	case sqliteBusy, sqliteLocked:
		return true
	default:
		return false
	}
}

// openBusyWindow bounds how long Open waits out a database another process
// currently holds locked before giving up.
var openBusyWindow = 10 * time.Second

// Open opens the database at path, creating parent directories and
// applying any pending migrations. It is safe to call on an existing
// database, and safe to call concurrently from several processes against
// the same path — including a brand-new one.
//
// A brand-new database is the contended case: every mk process starting at
// once finds no schema and tries to create it. migrate serializes the
// schema work itself, but simply acquiring a usable connection can still
// fail outright, because setting journal_mode=WAL takes a write lock and
// SQLite reports a lock it cannot take as an error. Retrying is what turns
// that error into a queue.
func Open(path string) (*Store, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, fmt.Errorf("%w: creating %s: %v", ErrDB, filepath.Dir(path), err)
	}

	deadline := time.Now().Add(openBusyWindow)
	for attempt := 0; ; attempt++ {
		s, err := openOnce(path)
		if err == nil {
			return s, nil
		}
		if !isBusy(err) || !time.Now().Before(deadline) {
			return nil, err
		}
		time.Sleep(retryBackoff(attempt))
	}
}

// retryBackoff returns a jittered delay that grows with the attempt count.
// The jitter matters: without it a batch of processes that collided once
// wakes up together and collides again.
func retryBackoff(attempt int) time.Duration {
	d := 5 * time.Millisecond << min(attempt, 5)
	return d + time.Duration(rand.Int63n(int64(d)))
}

// openOnce is one attempt at Open, with no retrying.
func openOnce(path string) (*Store, error) {
	dsn := path + "?_pragma=busy_timeout(5000)" +
		"&_pragma=journal_mode(WAL)" +
		"&_pragma=foreign_keys(1)"

	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("%w: opening %s: %w", ErrDB, path, err)
	}
	if err := db.Ping(); err != nil {
		db.Close()
		return nil, fmt.Errorf("%w: opening %s: %w", ErrDB, path, err)
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

// withImmediateTx runs fn inside a BEGIN IMMEDIATE transaction held on one
// dedicated connection, committing if fn returns nil and rolling back
// otherwise. what names the operation for the error messages.
//
// IMMEDIATE, not the deferred BEGIN that database/sql issues, and this is
// not a preference. A deferred transaction that reads before it writes
// takes a read lock and then has to upgrade it for the first write —
// and SQLite answers a lock upgrade it cannot grant with SQLITE_BUSY
// immediately, without consulting the busy handler. busy_timeout does not
// help, so the transaction simply fails whenever another writer is active.
// IMMEDIATE takes the write lock up front, which is the one case
// busy_timeout does cover: contending callers queue instead of failing.
//
// Every read-then-write transaction in this package must go through here.
func (s *Store) withImmediateTx(what string, fn func(ctx context.Context, conn *sql.Conn) error) error {
	ctx := context.Background()

	// One dedicated connection: BEGIN/COMMIT are connection state, so they
	// must not be scattered across the pool.
	conn, err := s.db.Conn(ctx)
	if err != nil {
		return fmt.Errorf("%w: %s: %w", ErrDB, what, err)
	}
	defer conn.Close()

	if _, err := conn.ExecContext(ctx, `BEGIN IMMEDIATE`); err != nil {
		return fmt.Errorf("%w: %s: %w", ErrDB, what, err)
	}
	// Deferred LIFO: this rollback runs before conn.Close above, so the
	// connection never returns to the pool inside a transaction.
	committed := false
	defer func() {
		if !committed {
			conn.ExecContext(ctx, `ROLLBACK`)
		}
	}()

	if err := fn(ctx, conn); err != nil {
		return err
	}

	if _, err := conn.ExecContext(ctx, `COMMIT`); err != nil {
		return fmt.Errorf("%w: %s: %w", ErrDB, what, err)
	}
	committed = true
	return nil
}

// migrate applies every embedded migration whose version exceeds the
// highest already recorded, in filename order.
//
// The whole read-then-apply sequence runs inside one transaction, which is
// what makes it safe across processes: a second mk process starting
// against the same brand-new database waits and then reads a
// schema_migrations that already names every migration, instead of also
// reading version 0 and also trying to create the schema — which fails
// with "table projects already exists".
func (s *Store) migrate() error {
	return s.withImmediateTx("running migrations", func(ctx context.Context, conn *sql.Conn) error {
		if _, err := conn.ExecContext(ctx,
			`CREATE TABLE IF NOT EXISTS schema_migrations (version INTEGER PRIMARY KEY)`); err != nil {
			return fmt.Errorf("%w: creating schema_migrations: %w", ErrDB, err)
		}

		var applied int
		if err := conn.QueryRowContext(ctx,
			`SELECT COALESCE(MAX(version), 0) FROM schema_migrations`).Scan(&applied); err != nil {
			return fmt.Errorf("%w: reading schema_migrations: %w", ErrDB, err)
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
			if _, err := conn.ExecContext(ctx, string(body)); err != nil {
				return fmt.Errorf("%w: applying migration %s: %w", ErrDB, name, err)
			}
			if _, err := conn.ExecContext(ctx,
				`INSERT INTO schema_migrations (version) VALUES (?)`, version); err != nil {
				return fmt.Errorf("%w: recording migration %s: %w", ErrDB, name, err)
			}
		}
		return nil
	})
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
