package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/khanhnguyendev/mind-knowledge/internal/model"
)

// EpicFields carries a partial update; nil pointers leave columns alone.
type EpicFields struct {
	ProjectID   *string
	Title       *string
	Description *string
	Status      *string
	Position    *int
}

const epicColumns = `id, project_id, title, description, status, position, created_at, updated_at`

func scanEpic(row interface{ Scan(...any) error }) (*model.Epic, error) {
	var e model.Epic
	err := row.Scan(&e.ID, &e.ProjectID, &e.Title, &e.Description,
		&e.Status, &e.Position, &e.CreatedAt, &e.UpdatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("%w: reading epic: %v", ErrDB, err)
	}
	return &e, nil
}

// nextPosition returns one past the highest position in a sibling set.
func (s *Store) nextPosition(table, parentColumn, parentID string) (int, error) {
	var maxPos sql.NullInt64
	err := s.db.QueryRow(
		fmt.Sprintf(`SELECT MAX(position) FROM %s WHERE %s = ?`, table, parentColumn),
		parentID).Scan(&maxPos)
	if err != nil {
		return 0, fmt.Errorf("%w: reading positions from %s: %v", ErrDB, table, err)
	}
	if !maxPos.Valid {
		return 0, nil
	}
	return int(maxPos.Int64) + 1, nil
}

// CreateEpic adds an epic to an existing project.
func (s *Store) CreateEpic(projectID, title, description string) (*model.Epic, error) {
	if title == "" {
		return nil, fmt.Errorf("%w: epic title is required", ErrInvalid)
	}
	project, err := s.GetProject(projectID)
	if err != nil {
		return nil, err
	}

	id, err := s.uniqueID("epics")
	if err != nil {
		return nil, err
	}
	pos, err := s.nextPosition("epics", "project_id", project.ID)
	if err != nil {
		return nil, err
	}
	now := timestamp()

	_, err = s.db.Exec(
		`INSERT INTO epics (`+epicColumns+`)
		 VALUES (?, ?, ?, ?, 'backlog', ?, ?, ?)`,
		id, project.ID, title, description, pos, now, now)
	if err != nil {
		// The only unique constraint on epics is its primary key, so
		// this is an id collision — bad input (2), exactly as
		// CreateProject and CreateWikiPage already classify it. Returning
		// raw ErrDB here made the same condition exit 3 on some commands
		// and 2 on others.
		if isUniqueViolation(err) {
			return nil, fmt.Errorf("%w: epic id %q is already taken", ErrInvalid, id)
		}
		return nil, fmt.Errorf("%w: inserting epic: %v", ErrDB, err)
	}
	return s.GetEpic(id)
}

// GetEpic looks an epic up by id.
func (s *Store) GetEpic(id string) (*model.Epic, error) {
	row := s.db.QueryRow(`SELECT `+epicColumns+` FROM epics WHERE id = ?`, id)
	e, err := scanEpic(row)
	if errors.Is(err, ErrNotFound) {
		return nil, fmt.Errorf("%w: no epic %q", ErrNotFound, id)
	}
	return e, err
}

// ListEpics returns epics ordered by position. Empty filters match all.
func (s *Store) ListEpics(projectID, status string, limit int) ([]model.Epic, error) {
	if status != "" && !model.ValidEpicStatus(status) {
		return nil, fmt.Errorf("%w: unknown epic status %q", ErrInvalid, status)
	}

	query := `SELECT ` + epicColumns + ` FROM epics WHERE 1 = 1`
	args := []any{}
	if projectID != "" {
		p, err := s.GetProject(projectID)
		if err != nil {
			return nil, err
		}
		query += ` AND project_id = ?`
		args = append(args, p.ID)
	}
	if status != "" {
		query += ` AND status = ?`
		args = append(args, status)
	}
	query += ` ORDER BY position, created_at`
	if limit > 0 {
		query += ` LIMIT ?`
		args = append(args, limit)
	}

	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("%w: listing epics: %v", ErrDB, err)
	}
	defer rows.Close()

	out := []model.Epic{}
	for rows.Next() {
		e, err := scanEpic(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *e)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("%w: listing epics: %v", ErrDB, err)
	}
	return out, nil
}

// UpdateEpic applies the non-nil fields.
func (s *Store) UpdateEpic(id string, f EpicFields) (*model.Epic, error) {
	current, err := s.GetEpic(id)
	if err != nil {
		return nil, err
	}
	if f.Status != nil && !model.ValidEpicStatus(*f.Status) {
		return nil, fmt.Errorf(
			"%w: unknown epic status %q (want backlog, in-progress, done, or dropped)",
			ErrInvalid, *f.Status)
	}
	if f.Title != nil && *f.Title == "" {
		return nil, fmt.Errorf("%w: epic title cannot be empty", ErrInvalid)
	}

	next := *current
	if f.Title != nil {
		next.Title = *f.Title
	}
	if f.Description != nil {
		next.Description = *f.Description
	}
	if f.Status != nil {
		next.Status = *f.Status
	}
	if f.Position != nil {
		next.Position = *f.Position
	}
	if f.ProjectID != nil {
		p, err := s.GetProject(*f.ProjectID)
		if err != nil {
			return nil, err
		}
		next.ProjectID = p.ID
		// If moving to a different project and position wasn't explicitly set,
		// recompute position so the epic lands at the end of its new siblings.
		if next.ProjectID != current.ProjectID && f.Position == nil {
			pos, err := s.nextPosition("epics", "project_id", next.ProjectID)
			if err != nil {
				return nil, err
			}
			next.Position = pos
		}
	}

	_, err = s.db.Exec(
		`UPDATE epics SET project_id = ?, title = ?, description = ?, status = ?,
		 position = ?, updated_at = ? WHERE id = ?`,
		next.ProjectID, next.Title, next.Description, next.Status,
		next.Position, timestamp(), current.ID)
	if err != nil {
		return nil, fmt.Errorf("%w: updating epic: %v", ErrDB, err)
	}
	return s.GetEpic(current.ID)
}

// DeleteEpic removes an epic; its stories cascade away in the database, so
// their links and tags are collected and cleaned here before the delete.
func (s *Store) DeleteEpic(id string) error {
	e, err := s.GetEpic(id)
	if err != nil {
		return err
	}

	// This transaction reads before it writes, so it must be IMMEDIATE:
	// see withImmediateTx.
	return s.withImmediateTx("deleting epic", func(ctx context.Context, conn *sql.Conn) error {
		storyIDs, err := txIDs(ctx, conn, `SELECT id FROM stories WHERE epic_id = ?`, e.ID)
		if err != nil {
			return err
		}
		if err := deleteEntityRefs(ctx, conn, "story", storyIDs); err != nil {
			return err
		}
		if err := deleteEntityRefs(ctx, conn, "epic", []string{e.ID}); err != nil {
			return err
		}
		if _, err := conn.ExecContext(ctx, `DELETE FROM epics WHERE id = ?`, e.ID); err != nil {
			return fmt.Errorf("%w: deleting epic: %v", ErrDB, err)
		}
		return nil
	})
}
