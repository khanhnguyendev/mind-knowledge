package store

import (
	"database/sql"
	"errors"
	"fmt"

	"github.com/khanhnguyendev/mind-knowledge/internal/model"
)

const logColumns = `id, ts, kind, COALESCE(project_id, ''), COALESCE(ref, ''), summary`

func scanLogEntry(row interface{ Scan(...any) error }) (*model.LogEntry, error) {
	var e model.LogEntry
	err := row.Scan(&e.ID, &e.TS, &e.Kind, &e.ProjectID, &e.Ref, &e.Summary)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("%w: reading log entry: %v", ErrDB, err)
	}
	return &e, nil
}

// AddLog appends an entry. The project and ref are optional; kind and
// summary are not.
func (s *Store) AddLog(kind, projectID, ref, summary string) (*model.LogEntry, error) {
	if kind == "" {
		return nil, fmt.Errorf("%w: log kind is required", ErrInvalid)
	}
	if summary == "" {
		return nil, fmt.Errorf("%w: log summary is required", ErrInvalid)
	}

	resolvedProject := ""
	if projectID != "" {
		p, err := s.GetProject(projectID)
		if err != nil {
			return nil, err
		}
		resolvedProject = p.ID
	}

	res, err := s.db.Exec(
		`INSERT INTO log (ts, kind, project_id, ref, summary) VALUES (?, ?, ?, ?, ?)`,
		timestamp(), kind, nullable(resolvedProject), nullable(ref), summary)
	if err != nil {
		return nil, fmt.Errorf("%w: inserting log entry: %v", ErrDB, err)
	}

	id, err := res.LastInsertId()
	if err != nil {
		return nil, fmt.Errorf("%w: reading log entry id: %v", ErrDB, err)
	}

	row := s.db.QueryRow(`SELECT `+logColumns+` FROM log WHERE id = ?`, id)
	return scanLogEntry(row)
}

// ListLog returns entries newest first.
func (s *Store) ListLog(kind, projectID string, limit int) ([]model.LogEntry, error) {
	query := `SELECT ` + logColumns + ` FROM log WHERE 1 = 1`
	args := []any{}

	if kind != "" {
		query += ` AND kind = ?`
		args = append(args, kind)
	}
	if projectID != "" {
		p, err := s.GetProject(projectID)
		if err != nil {
			return nil, err
		}
		query += ` AND project_id = ?`
		args = append(args, p.ID)
	}
	query += ` ORDER BY id DESC`
	if limit > 0 {
		query += ` LIMIT ?`
		args = append(args, limit)
	}

	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("%w: listing log: %v", ErrDB, err)
	}
	defer rows.Close()

	out := []model.LogEntry{}
	for rows.Next() {
		e, err := scanLogEntry(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *e)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("%w: listing log: %v", ErrDB, err)
	}
	return out, nil
}
