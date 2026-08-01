package store

import (
	"database/sql"
	"fmt"
	"strings"

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
	if relation != "" && !model.ValidRelation(relation) {
		return nil, fmt.Errorf(
			"%w: unknown relation %q (want derived-from, supersedes, references, or implements)",
			ErrInvalid, relation)
	}

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

// deleteEntityRefs removes every links row and entity_tags row naming any
// of ids under kind, on either end of the edge.
//
// links and entity_tags deliberately carry no foreign key to the entity
// they point at: their endpoints span five different tables, so no single
// REFERENCES clause could express the constraint. That makes cleanup this
// function's job, and it runs inside the caller's transaction so an entity
// and its references always disappear together. Skipping it leaves an edge
// that permanently vouches for the endpoint that survived — which silently
// disables wiki.orphans, wiki.uncited, and wiki.unprocessed for it.
func deleteEntityRefs(tx *sql.Tx, kind string, ids []string) error {
	if len(ids) == 0 {
		return nil
	}
	placeholders := strings.TrimSuffix(strings.Repeat("?, ", len(ids)), ", ")
	args := make([]any, 0, len(ids)+1)
	args = append(args, kind)
	for _, id := range ids {
		args = append(args, id)
	}

	for _, stmt := range []string{
		`DELETE FROM links WHERE from_kind = ? AND from_id IN (` + placeholders + `)`,
		`DELETE FROM links WHERE to_kind = ? AND to_id IN (` + placeholders + `)`,
		`DELETE FROM entity_tags WHERE entity_kind = ? AND entity_id IN (` + placeholders + `)`,
	} {
		if _, err := tx.Exec(stmt, args...); err != nil {
			return fmt.Errorf("%w: cleaning %s references: %v", ErrDB, kind, err)
		}
	}
	return nil
}

// deleteEntity removes one row from table together with every links and
// entity_tags row naming it, in a single transaction so a partial delete
// cannot leave the graph inconsistent.
func (s *Store) deleteEntity(kind, table, id string) error {
	tx, err := s.db.Begin()
	if err != nil {
		return fmt.Errorf("%w: deleting %s: %v", ErrDB, kind, err)
	}
	defer tx.Rollback()

	if err := deleteEntityRefs(tx, kind, []string{id}); err != nil {
		return err
	}
	if _, err := tx.Exec(fmt.Sprintf(`DELETE FROM %s WHERE id = ?`, table), id); err != nil {
		return fmt.Errorf("%w: deleting %s: %v", ErrDB, kind, err)
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("%w: deleting %s: %v", ErrDB, kind, err)
	}
	return nil
}

// txIDs collects the single-column ids a query returns, inside tx.
func txIDs(tx *sql.Tx, query string, args ...any) ([]string, error) {
	rows, err := tx.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("%w: collecting ids: %v", ErrDB, err)
	}
	defer rows.Close()

	out := []string{}
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("%w: reading id: %v", ErrDB, err)
		}
		out = append(out, id)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("%w: collecting ids: %v", ErrDB, err)
	}
	return out, nil
}

// entityTables maps an entity kind to the table that owns its ids.
var entityTables = map[string]string{
	"project": "projects",
	"epic":    "epics",
	"story":   "stories",
	"source":  "sources",
	"wiki":    "wiki_pages",
}

// idSet reads every id in a table.
func (s *Store) idSet(table string) (map[string]bool, error) {
	rows, err := s.db.Query(fmt.Sprintf(`SELECT id FROM %s`, table))
	if err != nil {
		return nil, fmt.Errorf("%w: reading ids from %s: %v", ErrDB, table, err)
	}
	defer rows.Close()

	out := map[string]bool{}
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("%w: reading id from %s: %v", ErrDB, table, err)
		}
		out[id] = true
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("%w: reading ids from %s: %v", ErrDB, table, err)
	}
	return out, nil
}

// DanglingLinks returns edges at least one of whose endpoints no longer
// names an existing entity. links carries no foreign keys — the endpoints
// span five tables, so no single constraint could express it — which means
// a database written by a binary that did not clean up after itself can
// still hold edges vouching for entities that are gone. An edge naming a
// kind that is not an entity kind at all counts as dangling too, since
// nothing could ever resolve it.
func (s *Store) DanglingLinks() ([]model.Link, error) {
	links, err := s.ListLinks("", "", "", "", "")
	if err != nil {
		return nil, err
	}

	known := make(map[string]map[string]bool, len(entityTables))
	for kind, table := range entityTables {
		ids, err := s.idSet(table)
		if err != nil {
			return nil, err
		}
		known[kind] = ids
	}

	out := []model.Link{}
	for _, l := range links {
		if !known[l.FromKind][l.FromID] || !known[l.ToKind][l.ToID] {
			out = append(out, l)
		}
	}
	return out, nil
}

// RemoveLink deletes one edge, reporting ErrNotFound when it was absent.
func (s *Store) RemoveLink(fromKind, fromID, toKind, toID, relation string) error {
	if !model.ValidRelation(relation) {
		return fmt.Errorf(
			"%w: unknown relation %q (want derived-from, supersedes, references, or implements)",
			ErrInvalid, relation)
	}

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
