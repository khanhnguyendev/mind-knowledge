package store

import (
	"database/sql"
	"errors"
	"fmt"

	"modernc.org/sqlite"

	"github.com/khanhnguyendev/mind-knowledge/internal/model"
)

// ProjectFields carries a partial update. A nil pointer leaves the column
// unchanged; a non-nil pointer sets it, including to the empty string.
type ProjectFields struct {
	Name      *string
	RepoPath  *string
	GitRemote *string
	Status    *string
}

const projectColumns = `id, name, repo_path, git_remote, status, created_at, updated_at`

// SQLite extended result codes for constraint violations. Hardcoded here
// rather than importing modernc.org/sqlite/lib, so this file's dependency
// surface stays at the two packages the project allows.
const (
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

func scanProject(row interface{ Scan(...any) error }) (*model.Project, error) {
	var p model.Project
	err := row.Scan(&p.ID, &p.Name, &p.RepoPath, &p.GitRemote,
		&p.Status, &p.CreatedAt, &p.UpdatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("%w: reading project: %v", ErrDB, err)
	}
	return &p, nil
}

// CreateProject registers a repository. Name must be non-empty and unique.
func (s *Store) CreateProject(name, repoPath, gitRemote string) (*model.Project, error) {
	if name == "" {
		return nil, fmt.Errorf("%w: project name is required", ErrInvalid)
	}
	if repoPath == "" {
		return nil, fmt.Errorf("%w: project repo path is required", ErrInvalid)
	}

	var existing string
	err := s.db.QueryRow(`SELECT id FROM projects WHERE name = ?`, name).Scan(&existing)
	if err == nil {
		return nil, fmt.Errorf("%w: project %q already exists as %s", ErrInvalid, name, existing)
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return nil, fmt.Errorf("%w: checking project name: %v", ErrDB, err)
	}

	id, err := s.uniqueID("projects")
	if err != nil {
		return nil, err
	}
	now := timestamp()

	_, err = s.db.Exec(
		`INSERT INTO projects (`+projectColumns+`) VALUES (?, ?, ?, ?, 'active', ?, ?)`,
		id, name, repoPath, gitRemote, now, now)
	if err != nil {
		// The SELECT-then-INSERT above is check-then-act and therefore
		// racy on its own; the table's UNIQUE(name) and PRIMARY KEY(id)
		// constraints are the real backstop. If either fires here
		// (concurrent CreateProject with the same name, or the
		// vanishingly unlikely id collision uniqueID didn't catch),
		// report it as bad input rather than leaking a raw driver error
		// classified as a database problem.
		if isUniqueViolation(err) {
			return nil, fmt.Errorf("%w: project %q already exists", ErrInvalid, name)
		}
		return nil, fmt.Errorf("%w: inserting project: %v", ErrDB, err)
	}

	return s.GetProject(id)
}

// GetProject resolves a project by id, falling back to an exact name match.
func (s *Store) GetProject(idOrName string) (*model.Project, error) {
	row := s.db.QueryRow(
		`SELECT `+projectColumns+` FROM projects WHERE id = ? OR name = ? LIMIT 1`,
		idOrName, idOrName)
	p, err := scanProject(row)
	if errors.Is(err, ErrNotFound) {
		return nil, fmt.Errorf("%w: no project %q", ErrNotFound, idOrName)
	}
	return p, err
}

// ListProjects returns projects ordered by name. An empty status returns
// every project; limit 0 returns all matches.
func (s *Store) ListProjects(status string, limit int) ([]model.Project, error) {
	if status != "" && !model.ValidProjectStatus(status) {
		return nil, fmt.Errorf("%w: unknown project status %q", ErrInvalid, status)
	}

	query := `SELECT ` + projectColumns + ` FROM projects`
	args := []any{}
	if status != "" {
		query += ` WHERE status = ?`
		args = append(args, status)
	}
	query += ` ORDER BY name`
	if limit > 0 {
		query += ` LIMIT ?`
		args = append(args, limit)
	}

	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("%w: listing projects: %v", ErrDB, err)
	}
	defer rows.Close()

	out := []model.Project{}
	for rows.Next() {
		p, err := scanProject(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *p)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("%w: listing projects: %v", ErrDB, err)
	}
	return out, nil
}

// UpdateProject applies the non-nil fields and returns the stored row.
func (s *Store) UpdateProject(id string, f ProjectFields) (*model.Project, error) {
	current, err := s.GetProject(id)
	if err != nil {
		return nil, err
	}
	if f.Status != nil && !model.ValidProjectStatus(*f.Status) {
		return nil, fmt.Errorf(
			"%w: unknown project status %q (want active, paused, or archived)",
			ErrInvalid, *f.Status)
	}
	if f.Name != nil && *f.Name == "" {
		return nil, fmt.Errorf("%w: project name cannot be empty", ErrInvalid)
	}

	next := *current
	if f.Name != nil {
		next.Name = *f.Name
	}
	if f.RepoPath != nil {
		next.RepoPath = *f.RepoPath
	}
	if f.GitRemote != nil {
		next.GitRemote = *f.GitRemote
	}
	if f.Status != nil {
		next.Status = *f.Status
	}

	_, err = s.db.Exec(
		`UPDATE projects SET name = ?, repo_path = ?, git_remote = ?, status = ?, updated_at = ?
		 WHERE id = ?`,
		next.Name, next.RepoPath, next.GitRemote, next.Status, timestamp(), current.ID)
	if err != nil {
		if isUniqueViolation(err) {
			return nil, fmt.Errorf("%w: project %q already exists", ErrInvalid, next.Name)
		}
		return nil, fmt.Errorf("%w: updating project: %v", ErrDB, err)
	}
	return s.GetProject(current.ID)
}

// DeleteProject removes a project. Its epics and stories cascade away.
func (s *Store) DeleteProject(id string) error {
	p, err := s.GetProject(id)
	if err != nil {
		return err
	}
	if _, err := s.db.Exec(`DELETE FROM projects WHERE id = ?`, p.ID); err != nil {
		return fmt.Errorf("%w: deleting project: %v", ErrDB, err)
	}
	return nil
}
