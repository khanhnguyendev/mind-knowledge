package store

import (
	"database/sql"
	"errors"
	"fmt"

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
// The two lookups are sequential and deliberately so: a single
// "WHERE id = ? OR name = ?" query would leave the id-first guarantee to
// whatever plan SQLite's optimizer happens to choose, which is not a
// property the schema or the query enforces. Doing the id lookup first in
// Go code makes that precedence explicit and immune to planner changes,
// table size, or SQLite version — load-bearing since nothing stops a
// project from being named after another project's id.
func (s *Store) GetProject(idOrName string) (*model.Project, error) {
	row := s.db.QueryRow(`SELECT `+projectColumns+` FROM projects WHERE id = ?`, idOrName)
	p, err := scanProject(row)
	if err == nil {
		return p, nil
	}
	if !errors.Is(err, ErrNotFound) {
		return nil, err
	}

	row = s.db.QueryRow(`SELECT `+projectColumns+` FROM projects WHERE name = ?`, idOrName)
	p, err = scanProject(row)
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

// DeleteProject removes a project. Its epics and stories cascade away in
// the database, so their links and tags are collected and cleaned here
// before the delete — otherwise the leak this guards against simply
// reappears one and two levels down.
func (s *Store) DeleteProject(id string) error {
	p, err := s.GetProject(id)
	if err != nil {
		return err
	}

	tx, err := s.db.Begin()
	if err != nil {
		return fmt.Errorf("%w: deleting project: %v", ErrDB, err)
	}
	defer tx.Rollback()

	epicIDs, err := txIDs(tx, `SELECT id FROM epics WHERE project_id = ?`, p.ID)
	if err != nil {
		return err
	}
	storyIDs, err := txIDs(tx,
		`SELECT s.id FROM stories s JOIN epics e ON e.id = s.epic_id
		 WHERE e.project_id = ?`, p.ID)
	if err != nil {
		return err
	}

	if err := deleteEntityRefs(tx, "story", storyIDs); err != nil {
		return err
	}
	if err := deleteEntityRefs(tx, "epic", epicIDs); err != nil {
		return err
	}
	if err := deleteEntityRefs(tx, "project", []string{p.ID}); err != nil {
		return err
	}
	if _, err := tx.Exec(`DELETE FROM projects WHERE id = ?`, p.ID); err != nil {
		return fmt.Errorf("%w: deleting project: %v", ErrDB, err)
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("%w: deleting project: %v", ErrDB, err)
	}
	return nil
}
