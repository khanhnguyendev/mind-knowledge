package store

import (
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"github.com/khanhnguyendev/mind-knowledge/internal/model"
)

// StoryFields carries a partial update; nil pointers leave columns alone.
// Notes and AppendNotes are mutually exclusive.
type StoryFields struct {
	EpicID      *string
	Title       *string
	Description *string
	Status      *string
	Priority    *string
	Position    *int
	Acceptance  *string
	Plan        *string
	Notes       *string
	AppendNotes *string
}

// StoryFilter narrows a listing. Empty strings match everything; Limit 0
// returns all matches. ProjectID and EpicID may both be set, in which case
// both must match.
type StoryFilter struct {
	ProjectID string
	EpicID    string
	Status    string
	Priority  string
	Limit     int
}

const storyColumns = `id, epic_id, title, description, status, priority, position,
	acceptance, plan, notes, created_at, updated_at`

func scanStory(row interface{ Scan(...any) error }) (*model.Story, error) {
	var s model.Story
	err := row.Scan(&s.ID, &s.EpicID, &s.Title, &s.Description, &s.Status,
		&s.Priority, &s.Position, &s.Acceptance, &s.Plan, &s.Notes,
		&s.CreatedAt, &s.UpdatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("%w: reading story: %v", ErrDB, err)
	}
	return &s, nil
}

// CreateStory adds a story to an existing epic.
func (s *Store) CreateStory(epicID, title, description string) (*model.Story, error) {
	if title == "" {
		return nil, fmt.Errorf("%w: story title is required", ErrInvalid)
	}
	epic, err := s.GetEpic(epicID)
	if err != nil {
		return nil, err
	}

	id, err := s.uniqueID("stories")
	if err != nil {
		return nil, err
	}
	pos, err := s.nextPosition("stories", "epic_id", epic.ID)
	if err != nil {
		return nil, err
	}
	now := timestamp()

	_, err = s.db.Exec(
		`INSERT INTO stories (`+storyColumns+`)
		 VALUES (?, ?, ?, ?, 'backlog', 'med', ?, '', '', '', ?, ?)`,
		id, epic.ID, title, description, pos, now, now)
	if err != nil {
		// The only unique constraint on stories is its primary key, so
		// this is an id collision — bad input (2), exactly as
		// CreateProject and CreateWikiPage already classify it. Returning
		// raw ErrDB here made the same condition exit 3 on some commands
		// and 2 on others.
		if isUniqueViolation(err) {
			return nil, fmt.Errorf("%w: story id %q is already taken", ErrInvalid, id)
		}
		return nil, fmt.Errorf("%w: inserting story: %v", ErrDB, err)
	}
	return s.GetStory(id)
}

// GetStory looks a story up by id.
func (s *Store) GetStory(id string) (*model.Story, error) {
	row := s.db.QueryRow(`SELECT `+storyColumns+` FROM stories WHERE id = ?`, id)
	st, err := scanStory(row)
	if errors.Is(err, ErrNotFound) {
		return nil, fmt.Errorf("%w: no story %q", ErrNotFound, id)
	}
	return st, err
}

// ListStories returns stories ordered by position within their epic.
func (s *Store) ListStories(f StoryFilter) ([]model.Story, error) {
	if f.Status != "" && !model.ValidStoryStatus(f.Status) {
		return nil, fmt.Errorf("%w: unknown story status %q", ErrInvalid, f.Status)
	}
	if f.Priority != "" && !model.ValidPriority(f.Priority) {
		return nil, fmt.Errorf("%w: unknown priority %q", ErrInvalid, f.Priority)
	}

	query := `SELECT ` + prefixColumns(storyColumns, "s") + `
		FROM stories s JOIN epics e ON e.id = s.epic_id WHERE 1 = 1`
	args := []any{}

	if f.ProjectID != "" {
		p, err := s.GetProject(f.ProjectID)
		if err != nil {
			return nil, err
		}
		query += ` AND e.project_id = ?`
		args = append(args, p.ID)
	}
	if f.EpicID != "" {
		epic, err := s.GetEpic(f.EpicID)
		if err != nil {
			return nil, err
		}
		query += ` AND s.epic_id = ?`
		args = append(args, epic.ID)
	}
	if f.Status != "" {
		query += ` AND s.status = ?`
		args = append(args, f.Status)
	}
	if f.Priority != "" {
		query += ` AND s.priority = ?`
		args = append(args, f.Priority)
	}

	query += ` ORDER BY e.position, s.position, s.created_at`
	if f.Limit > 0 {
		query += ` LIMIT ?`
		args = append(args, f.Limit)
	}

	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("%w: listing stories: %v", ErrDB, err)
	}
	defer rows.Close()

	out := []model.Story{}
	for rows.Next() {
		st, err := scanStory(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *st)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("%w: listing stories: %v", ErrDB, err)
	}
	return out, nil
}

// prefixColumns qualifies a comma-separated column list with a table alias
// so it can be used in a join.
func prefixColumns(columns, alias string) string {
	parts := strings.Split(columns, ",")
	for i, p := range parts {
		parts[i] = alias + "." + strings.TrimSpace(p)
	}
	return strings.Join(parts, ", ")
}

// UpdateStory applies the non-nil fields. It validates enum membership but
// never checks whether a transition is workflow-legal.
func (s *Store) UpdateStory(id string, f StoryFields) (*model.Story, error) {
	current, err := s.GetStory(id)
	if err != nil {
		return nil, err
	}
	if f.Notes != nil && f.AppendNotes != nil {
		return nil, fmt.Errorf("%w: set notes or append to them, not both", ErrInvalid)
	}
	if f.Status != nil && !model.ValidStoryStatus(*f.Status) {
		return nil, fmt.Errorf(
			"%w: unknown story status %q (want backlog, ready, in-progress, review, done, or dropped)",
			ErrInvalid, *f.Status)
	}
	if f.Priority != nil && !model.ValidPriority(*f.Priority) {
		return nil, fmt.Errorf(
			"%w: unknown priority %q (want low, med, or high)", ErrInvalid, *f.Priority)
	}
	if f.Title != nil && *f.Title == "" {
		return nil, fmt.Errorf("%w: story title cannot be empty", ErrInvalid)
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
	if f.Priority != nil {
		next.Priority = *f.Priority
	}
	if f.Position != nil {
		next.Position = *f.Position
	}
	if f.Acceptance != nil {
		next.Acceptance = *f.Acceptance
	}
	if f.Plan != nil {
		next.Plan = *f.Plan
	}
	if f.Notes != nil {
		next.Notes = *f.Notes
	}
	if f.AppendNotes != nil {
		if next.Notes == "" {
			next.Notes = *f.AppendNotes
		} else {
			next.Notes = next.Notes + "\n\n" + *f.AppendNotes
		}
	}
	if f.EpicID != nil {
		epic, err := s.GetEpic(*f.EpicID)
		if err != nil {
			return nil, err
		}
		next.EpicID = epic.ID
		// If moving to a different epic and position wasn't explicitly set,
		// recompute position so the story lands at the end of its new siblings.
		if next.EpicID != current.EpicID && f.Position == nil {
			pos, err := s.nextPosition("stories", "epic_id", next.EpicID)
			if err != nil {
				return nil, err
			}
			next.Position = pos
		}
	}

	_, err = s.db.Exec(
		`UPDATE stories SET epic_id = ?, title = ?, description = ?, status = ?,
		 priority = ?, position = ?, acceptance = ?, plan = ?, notes = ?, updated_at = ?
		 WHERE id = ?`,
		next.EpicID, next.Title, next.Description, next.Status, next.Priority,
		next.Position, next.Acceptance, next.Plan, next.Notes, timestamp(), current.ID)
	if err != nil {
		return nil, fmt.Errorf("%w: updating story: %v", ErrDB, err)
	}
	return s.GetStory(current.ID)
}

// DeleteStory removes a story together with its links and tags.
func (s *Store) DeleteStory(id string) error {
	st, err := s.GetStory(id)
	if err != nil {
		return err
	}
	return s.deleteEntity("story", "stories", st.ID)
}
