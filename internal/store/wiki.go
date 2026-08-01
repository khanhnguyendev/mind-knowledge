package store

import (
	"database/sql"
	"errors"
	"fmt"

	"github.com/khanhnguyendev/mind-knowledge/internal/model"
)

// WikiFields carries a partial update; nil pointers leave columns alone.
type WikiFields struct {
	Slug      *string
	Title     *string
	Summary   *string
	Kind      *string
	Body      *string
	Status    *string
	ProjectID *string
}

const wikiColumns = `id, slug, title, summary, kind, body, status,
	COALESCE(project_id, ''), created_at, updated_at`

func scanWikiPage(row interface{ Scan(...any) error }) (*model.WikiPage, error) {
	var p model.WikiPage
	err := row.Scan(&p.ID, &p.Slug, &p.Title, &p.Summary, &p.Kind,
		&p.Body, &p.Status, &p.ProjectID, &p.CreatedAt, &p.UpdatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("%w: reading wiki page: %v", ErrDB, err)
	}
	return &p, nil
}

// nullable converts an empty string to a SQL NULL so the project_id
// foreign key stays satisfiable for cross-project pages.
func nullable(v string) any {
	if v == "" {
		return nil
	}
	return v
}

// CreateWikiPage adds a page. An empty slug is derived from the title.
func (s *Store) CreateWikiPage(slug, title, kind, summary, body, projectID string) (*model.WikiPage, error) {
	if title == "" {
		return nil, fmt.Errorf("%w: wiki page title is required", ErrInvalid)
	}
	if kind == "" {
		kind = "concept"
	}
	if !model.ValidWikiKind(kind) {
		return nil, fmt.Errorf(
			"%w: unknown wiki kind %q (want summary, concept, entity, decision, spec, synthesis, or comparison)",
			ErrInvalid, kind)
	}

	if slug == "" {
		slug = title
	}
	slug = model.Slugify(slug)
	if slug == "" {
		return nil, fmt.Errorf("%w: could not derive a slug from %q", ErrInvalid, title)
	}

	var existing string
	err := s.db.QueryRow(`SELECT id FROM wiki_pages WHERE slug = ?`, slug).Scan(&existing)
	if err == nil {
		return nil, fmt.Errorf("%w: wiki page %q already exists as %s", ErrInvalid, slug, existing)
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return nil, fmt.Errorf("%w: checking wiki slug: %v", ErrDB, err)
	}

	resolvedProject := ""
	if projectID != "" {
		p, err := s.GetProject(projectID)
		if err != nil {
			return nil, err
		}
		resolvedProject = p.ID
	}

	id, err := s.uniqueID("wiki_pages")
	if err != nil {
		return nil, err
	}
	now := timestamp()

	_, err = s.db.Exec(
		`INSERT INTO wiki_pages
		 (id, slug, title, summary, kind, body, status, project_id, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?, 'current', ?, ?, ?)`,
		id, slug, title, summary, kind, body, nullable(resolvedProject), now, now)
	if err != nil {
		if isUniqueViolation(err) {
			return nil, fmt.Errorf("%w: wiki page %q already exists", ErrInvalid, slug)
		}
		return nil, fmt.Errorf("%w: inserting wiki page: %v", ErrDB, err)
	}
	return s.GetWikiPage(id)
}

// GetWikiPage resolves a page by id, falling back to an exact slug match.
// The two lookups are sequential and deliberately so: a single
// "WHERE id = ? OR slug = ?" query would leave the id-first guarantee to
// whatever plan SQLite's optimizer happens to choose, which is not a
// property the schema or the query enforces. Doing the id lookup first in
// Go code makes that precedence explicit and immune to planner changes,
// table size, or SQLite version — load-bearing since nothing stops a page
// from being slugged after another page's id.
func (s *Store) GetWikiPage(idOrSlug string) (*model.WikiPage, error) {
	row := s.db.QueryRow(`SELECT `+wikiColumns+` FROM wiki_pages WHERE id = ?`, idOrSlug)
	p, err := scanWikiPage(row)
	if err == nil {
		return p, nil
	}
	if !errors.Is(err, ErrNotFound) {
		return nil, err
	}

	row = s.db.QueryRow(`SELECT `+wikiColumns+` FROM wiki_pages WHERE slug = ?`, idOrSlug)
	p, err = scanWikiPage(row)
	if errors.Is(err, ErrNotFound) {
		return nil, fmt.Errorf("%w: no wiki page %q", ErrNotFound, idOrSlug)
	}
	return p, err
}

// ListWikiPages returns pages ordered by kind then slug.
func (s *Store) ListWikiPages(kind, status, projectID string, limit int) ([]model.WikiPage, error) {
	if kind != "" && !model.ValidWikiKind(kind) {
		return nil, fmt.Errorf("%w: unknown wiki kind %q", ErrInvalid, kind)
	}
	if status != "" && !model.ValidWikiStatus(status) {
		return nil, fmt.Errorf("%w: unknown wiki status %q", ErrInvalid, status)
	}

	query := `SELECT ` + wikiColumns + ` FROM wiki_pages WHERE 1 = 1`
	args := []any{}
	if kind != "" {
		query += ` AND kind = ?`
		args = append(args, kind)
	}
	if status != "" {
		query += ` AND status = ?`
		args = append(args, status)
	}
	if projectID != "" {
		p, err := s.GetProject(projectID)
		if err != nil {
			return nil, err
		}
		query += ` AND project_id = ?`
		args = append(args, p.ID)
	}
	query += ` ORDER BY kind, slug`
	if limit > 0 {
		query += ` LIMIT ?`
		args = append(args, limit)
	}

	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("%w: listing wiki pages: %v", ErrDB, err)
	}
	defer rows.Close()

	out := []model.WikiPage{}
	for rows.Next() {
		p, err := scanWikiPage(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *p)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("%w: listing wiki pages: %v", ErrDB, err)
	}
	return out, nil
}

// UpdateWikiPage applies the non-nil fields.
func (s *Store) UpdateWikiPage(idOrSlug string, f WikiFields) (*model.WikiPage, error) {
	current, err := s.GetWikiPage(idOrSlug)
	if err != nil {
		return nil, err
	}
	if f.Kind != nil && !model.ValidWikiKind(*f.Kind) {
		return nil, fmt.Errorf("%w: unknown wiki kind %q", ErrInvalid, *f.Kind)
	}
	if f.Status != nil && !model.ValidWikiStatus(*f.Status) {
		return nil, fmt.Errorf(
			"%w: unknown wiki status %q (want current, stale, or superseded)",
			ErrInvalid, *f.Status)
	}
	if f.Title != nil && *f.Title == "" {
		return nil, fmt.Errorf("%w: wiki page title cannot be empty", ErrInvalid)
	}

	next := *current
	if f.Title != nil {
		next.Title = *f.Title
	}
	if f.Summary != nil {
		next.Summary = *f.Summary
	}
	if f.Kind != nil {
		next.Kind = *f.Kind
	}
	if f.Body != nil {
		next.Body = *f.Body
	}
	if f.Status != nil {
		next.Status = *f.Status
	}
	if f.Slug != nil {
		slug := model.Slugify(*f.Slug)
		if slug == "" {
			return nil, fmt.Errorf("%w: slug cannot be empty", ErrInvalid)
		}
		next.Slug = slug
	}
	if f.ProjectID != nil {
		if *f.ProjectID == "" {
			next.ProjectID = ""
		} else {
			p, err := s.GetProject(*f.ProjectID)
			if err != nil {
				return nil, err
			}
			next.ProjectID = p.ID
		}
	}

	_, err = s.db.Exec(
		`UPDATE wiki_pages SET slug = ?, title = ?, summary = ?, kind = ?, body = ?,
		 status = ?, project_id = ?, updated_at = ? WHERE id = ?`,
		next.Slug, next.Title, next.Summary, next.Kind, next.Body,
		next.Status, nullable(next.ProjectID), timestamp(), current.ID)
	if err != nil {
		if isUniqueViolation(err) {
			return nil, fmt.Errorf("%w: wiki page %q already exists", ErrInvalid, next.Slug)
		}
		return nil, fmt.Errorf("%w: updating wiki page: %v", ErrDB, err)
	}
	return s.GetWikiPage(current.ID)
}

// DeleteWikiPage removes a page.
func (s *Store) DeleteWikiPage(idOrSlug string) error {
	p, err := s.GetWikiPage(idOrSlug)
	if err != nil {
		return err
	}
	if _, err := s.db.Exec(`DELETE FROM wiki_pages WHERE id = ?`, p.ID); err != nil {
		return fmt.Errorf("%w: deleting wiki page: %v", ErrDB, err)
	}
	return nil
}
