package store

import (
	"fmt"

	"github.com/khanhnguyendev/mind-knowledge/internal/model"
)

// resolveEntity maps an entity kind and a user-supplied reference (an id,
// a project name, or a wiki slug) to the canonical id stored in links.
func (s *Store) resolveEntity(kind, ref string) (string, error) {
	if !model.ValidEntityKind(kind) {
		return "", fmt.Errorf(
			"%w: unknown entity kind %q (want project, epic, story, source, or wiki)",
			ErrInvalid, kind)
	}
	if ref == "" {
		return "", fmt.Errorf("%w: %s reference is required", ErrInvalid, kind)
	}

	switch kind {
	case "project":
		p, err := s.GetProject(ref)
		if err != nil {
			return "", err
		}
		return p.ID, nil
	case "epic":
		e, err := s.GetEpic(ref)
		if err != nil {
			return "", err
		}
		return e.ID, nil
	case "story":
		st, err := s.GetStory(ref)
		if err != nil {
			return "", err
		}
		return st.ID, nil
	case "source":
		src, err := s.GetSource(ref)
		if err != nil {
			return "", err
		}
		return src.ID, nil
	default: // wiki
		p, err := s.GetWikiPage(ref)
		if err != nil {
			return "", err
		}
		return p.ID, nil
	}
}

// ResolveEntity exposes entity resolution so callers can canonicalize a
// user-supplied reference before filtering.
func (s *Store) ResolveEntity(kind, ref string) (string, error) {
	return s.resolveEntity(kind, ref)
}

// AddLink records an edge between two entities. Adding the same edge twice
// is a no-op rather than an error, so a skill can re-run an ingest safely.
func (s *Store) AddLink(fromKind, fromID, toKind, toID, relation string) (*model.Link, error) {
	if !model.ValidRelation(relation) {
		return nil, fmt.Errorf(
			"%w: unknown relation %q (want derived-from, supersedes, references, or implements)",
			ErrInvalid, relation)
	}

	resolvedFrom, err := s.resolveEntity(fromKind, fromID)
	if err != nil {
		return nil, err
	}
	resolvedTo, err := s.resolveEntity(toKind, toID)
	if err != nil {
		return nil, err
	}

	_, err = s.db.Exec(
		`INSERT OR IGNORE INTO links (from_kind, from_id, to_kind, to_id, relation)
		 VALUES (?, ?, ?, ?, ?)`,
		fromKind, resolvedFrom, toKind, resolvedTo, relation)
	if err != nil {
		return nil, fmt.Errorf("%w: inserting link: %v", ErrDB, err)
	}

	return &model.Link{
		FromKind: fromKind,
		FromID:   resolvedFrom,
		ToKind:   toKind,
		ToID:     resolvedTo,
		Relation: relation,
	}, nil
}

// ListLinks returns edges matching every non-empty filter.
func (s *Store) ListLinks(fromKind, fromID, toKind, toID, relation string) ([]model.Link, error) {
	query := `SELECT from_kind, from_id, to_kind, to_id, relation FROM links WHERE 1 = 1`
	args := []any{}

	if fromKind != "" {
		query += ` AND from_kind = ?`
		args = append(args, fromKind)
	}
	if fromID != "" {
		query += ` AND from_id = ?`
		args = append(args, fromID)
	}
	if toKind != "" {
		query += ` AND to_kind = ?`
		args = append(args, toKind)
	}
	if toID != "" {
		query += ` AND to_id = ?`
		args = append(args, toID)
	}
	if relation != "" {
		query += ` AND relation = ?`
		args = append(args, relation)
	}
	query += ` ORDER BY from_kind, from_id, relation, to_kind, to_id`

	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("%w: listing links: %v", ErrDB, err)
	}
	defer rows.Close()

	out := []model.Link{}
	for rows.Next() {
		var l model.Link
		if err := rows.Scan(&l.FromKind, &l.FromID, &l.ToKind, &l.ToID, &l.Relation); err != nil {
			return nil, fmt.Errorf("%w: reading link: %v", ErrDB, err)
		}
		out = append(out, l)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("%w: listing links: %v", ErrDB, err)
	}
	return out, nil
}

// RemoveLink deletes one edge, reporting ErrNotFound when it was absent.
func (s *Store) RemoveLink(fromKind, fromID, toKind, toID, relation string) error {
	resolvedFrom, err := s.resolveEntity(fromKind, fromID)
	if err != nil {
		return err
	}
	resolvedTo, err := s.resolveEntity(toKind, toID)
	if err != nil {
		return err
	}

	res, err := s.db.Exec(
		`DELETE FROM links WHERE from_kind = ? AND from_id = ? AND to_kind = ?
		 AND to_id = ? AND relation = ?`,
		fromKind, resolvedFrom, toKind, resolvedTo, relation)
	if err != nil {
		return fmt.Errorf("%w: deleting link: %v", ErrDB, err)
	}

	affected, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("%w: deleting link: %v", ErrDB, err)
	}
	if affected == 0 {
		return fmt.Errorf("%w: no %s link from %s to %s", ErrNotFound, relation, fromID, toID)
	}
	return nil
}
