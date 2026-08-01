package store

import (
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"

	"github.com/khanhnguyendev/mind-knowledge/internal/model"
)

const sourceColumns = `id, uri, title, kind, body, asset_path, content_hash, ingested_at`

func scanSource(row interface{ Scan(...any) error }) (*model.Source, error) {
	var s model.Source
	err := row.Scan(&s.ID, &s.URI, &s.Title, &s.Kind, &s.Body,
		&s.AssetPath, &s.ContentHash, &s.IngestedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("%w: reading source: %v", ErrDB, err)
	}
	return &s, nil
}

// hashBody returns the hex SHA-256 of body, used to spot a source that has
// already been captured.
func hashBody(body string) string {
	sum := sha256.Sum256([]byte(body))
	return hex.EncodeToString(sum[:])
}

// HashBody exposes the content hash used for duplicate detection.
func HashBody(body string) string { return hashBody(body) }

// CreateSource captures a raw source. Sources are immutable: there is no
// update method, because layer 1 of the wiki is the source of truth.
func (s *Store) CreateSource(uri, title, kind, body, assetPath string) (*model.Source, error) {
	if title == "" {
		return nil, fmt.Errorf("%w: source title is required", ErrInvalid)
	}
	if kind == "" {
		kind = "note"
	}
	if !model.ValidSourceKind(kind) {
		return nil, fmt.Errorf(
			"%w: unknown source kind %q (want article, paper, transcript, chapter, asset, or note)",
			ErrInvalid, kind)
	}
	if body == "" && assetPath == "" {
		return nil, fmt.Errorf(
			"%w: a source needs a body or an asset path; mk does not fetch over the network",
			ErrInvalid)
	}

	id, err := s.uniqueID("sources")
	if err != nil {
		return nil, err
	}

	_, err = s.db.Exec(
		`INSERT INTO sources (`+sourceColumns+`) VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		id, uri, title, kind, body, assetPath, hashBody(body), timestamp())
	if err != nil {
		// The only unique constraint on sources is its primary key, so
		// this is an id collision — bad input (2), exactly as
		// CreateProject and CreateWikiPage already classify it. Returning
		// raw ErrDB here made the same condition exit 3 on some commands
		// and 2 on others.
		if isUniqueViolation(err) {
			return nil, fmt.Errorf("%w: source id %q is already taken", ErrInvalid, id)
		}
		return nil, fmt.Errorf("%w: inserting source: %v", ErrDB, err)
	}
	return s.GetSource(id)
}

// GetSource looks a source up by id.
func (s *Store) GetSource(id string) (*model.Source, error) {
	row := s.db.QueryRow(`SELECT `+sourceColumns+` FROM sources WHERE id = ?`, id)
	src, err := scanSource(row)
	if errors.Is(err, ErrNotFound) {
		return nil, fmt.Errorf("%w: no source %q", ErrNotFound, id)
	}
	return src, err
}

// FindSourceByHash returns the source whose body hashes to hash, so a
// caller can avoid ingesting the same document twice.
func (s *Store) FindSourceByHash(hash string) (*model.Source, error) {
	row := s.db.QueryRow(
		`SELECT `+sourceColumns+` FROM sources WHERE content_hash = ? LIMIT 1`, hash)
	src, err := scanSource(row)
	if errors.Is(err, ErrNotFound) {
		return nil, fmt.Errorf("%w: no source with hash %q", ErrNotFound, hash)
	}
	return src, err
}

// ListSources returns sources newest first.
func (s *Store) ListSources(kind string, limit int) ([]model.Source, error) {
	if kind != "" && !model.ValidSourceKind(kind) {
		return nil, fmt.Errorf("%w: unknown source kind %q", ErrInvalid, kind)
	}

	query := `SELECT ` + sourceColumns + ` FROM sources`
	args := []any{}
	if kind != "" {
		query += ` WHERE kind = ?`
		args = append(args, kind)
	}
	query += ` ORDER BY ingested_at DESC, id`
	if limit > 0 {
		query += ` LIMIT ?`
		args = append(args, limit)
	}

	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("%w: listing sources: %v", ErrDB, err)
	}
	defer rows.Close()

	out := []model.Source{}
	for rows.Next() {
		src, err := scanSource(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *src)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("%w: listing sources: %v", ErrDB, err)
	}
	return out, nil
}

// DeleteSource removes a source together with its links and tags.
func (s *Store) DeleteSource(id string) error {
	src, err := s.GetSource(id)
	if err != nil {
		return err
	}
	return s.deleteEntity("source", "sources", src.ID)
}
