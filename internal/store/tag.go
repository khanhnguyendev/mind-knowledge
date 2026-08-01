package store

import (
	"fmt"
	"strings"

	"github.com/khanhnguyendev/mind-knowledge/internal/model"
)

// AddTag attaches a tag to an entity, creating the tag if it is new.
// Attaching the same tag twice is a no-op.
func (s *Store) AddTag(name, entityKind, entityID string) error {
	name = strings.TrimSpace(strings.ToLower(name))
	if name == "" {
		return fmt.Errorf("%w: tag name is required", ErrInvalid)
	}

	resolved, err := s.resolveEntity(entityKind, entityID)
	if err != nil {
		return err
	}

	if _, err := s.db.Exec(
		`INSERT OR IGNORE INTO tags (name) VALUES (?)`, name); err != nil {
		return fmt.Errorf("%w: inserting tag: %v", ErrDB, err)
	}
	if _, err := s.db.Exec(
		`INSERT OR IGNORE INTO entity_tags (tag, entity_kind, entity_id) VALUES (?, ?, ?)`,
		name, entityKind, resolved); err != nil {
		return fmt.Errorf("%w: attaching tag: %v", ErrDB, err)
	}
	return nil
}

// ListTags returns every tag name currently attached to something.
func (s *Store) ListTags() ([]string, error) {
	rows, err := s.db.Query(
		`SELECT DISTINCT tag FROM entity_tags ORDER BY tag`)
	if err != nil {
		return nil, fmt.Errorf("%w: listing tags: %v", ErrDB, err)
	}
	defer rows.Close()

	out := []string{}
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, fmt.Errorf("%w: reading tag: %v", ErrDB, err)
		}
		out = append(out, name)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("%w: listing tags: %v", ErrDB, err)
	}
	return out, nil
}

// TagsFor returns the tags attached to one entity.
func (s *Store) TagsFor(entityKind, entityID string) ([]string, error) {
	resolved, err := s.resolveEntity(entityKind, entityID)
	if err != nil {
		return nil, err
	}

	rows, err := s.db.Query(
		`SELECT tag FROM entity_tags WHERE entity_kind = ? AND entity_id = ? ORDER BY tag`,
		entityKind, resolved)
	if err != nil {
		return nil, fmt.Errorf("%w: reading tags: %v", ErrDB, err)
	}
	defer rows.Close()

	out := []string{}
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, fmt.Errorf("%w: reading tag: %v", ErrDB, err)
		}
		out = append(out, name)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("%w: reading tags: %v", ErrDB, err)
	}
	return out, nil
}

// TaggedWith returns everything carrying a tag. It reuses model.Link so
// the JSON shape stays familiar: FromKind and FromID name the entity, and
// the target is the tag itself.
func (s *Store) TaggedWith(name string) ([]model.Link, error) {
	name = strings.TrimSpace(strings.ToLower(name))
	if name == "" {
		return nil, fmt.Errorf("%w: tag name is required", ErrInvalid)
	}

	rows, err := s.db.Query(
		`SELECT entity_kind, entity_id FROM entity_tags WHERE tag = ?
		 ORDER BY entity_kind, entity_id`, name)
	if err != nil {
		return nil, fmt.Errorf("%w: listing tagged entities: %v", ErrDB, err)
	}
	defer rows.Close()

	out := []model.Link{}
	for rows.Next() {
		var kind, id string
		if err := rows.Scan(&kind, &id); err != nil {
			return nil, fmt.Errorf("%w: reading tagged entity: %v", ErrDB, err)
		}
		out = append(out, model.Link{
			FromKind: kind,
			FromID:   id,
			ToKind:   "tag",
			ToID:     name,
			Relation: "tag",
		})
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("%w: listing tagged entities: %v", ErrDB, err)
	}
	return out, nil
}

// RemoveTag detaches a tag from an entity.
func (s *Store) RemoveTag(name, entityKind, entityID string) error {
	name = strings.TrimSpace(strings.ToLower(name))
	resolved, err := s.resolveEntity(entityKind, entityID)
	if err != nil {
		return err
	}

	res, err := s.db.Exec(
		`DELETE FROM entity_tags WHERE tag = ? AND entity_kind = ? AND entity_id = ?`,
		name, entityKind, resolved)
	if err != nil {
		return fmt.Errorf("%w: detaching tag: %v", ErrDB, err)
	}
	affected, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("%w: detaching tag: %v", ErrDB, err)
	}
	if affected == 0 {
		return fmt.Errorf("%w: %s is not tagged %q", ErrNotFound, entityID, name)
	}
	return nil
}
