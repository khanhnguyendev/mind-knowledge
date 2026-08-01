package store

import (
	"fmt"
	"strings"
)

// SearchHit is one full-text match. Score comes from SQLite's bm25(),
// where a lower value is a better match.
type SearchHit struct {
	Kind    string  `json:"kind"`
	ID      string  `json:"id"`
	Title   string  `json:"title"`
	Snippet string  `json:"snippet"`
	Score   float64 `json:"score"`
}

// Search runs a BM25 query across stories, wiki pages, and sources. An
// empty kinds slice searches everything; limit 0 returns all matches.
func (s *Store) Search(query string, kinds []string, limit int) ([]SearchHit, error) {
	trimmed := strings.TrimSpace(query)
	if trimmed == "" {
		return nil, fmt.Errorf("%w: search query is required", ErrInvalid)
	}

	for _, kind := range kinds {
		if kind != "story" && kind != "wiki" && kind != "source" {
			return nil, fmt.Errorf(
				"%w: unknown search kind %q (want story, wiki, or source)", ErrInvalid, kind)
		}
	}

	sql := `SELECT kind, ref_id, title,
			snippet(search_index, 3, '[', ']', '…', 12) AS snip,
			bm25(search_index) AS score
		FROM search_index
		WHERE search_index MATCH ?`
	args := []any{ftsQuery(trimmed)}

	if len(kinds) > 0 {
		placeholders := strings.TrimSuffix(strings.Repeat("?, ", len(kinds)), ", ")
		sql += ` AND kind IN (` + placeholders + `)`
		for _, kind := range kinds {
			args = append(args, kind)
		}
	}

	sql += ` ORDER BY score`
	if limit > 0 {
		sql += ` LIMIT ?`
		args = append(args, limit)
	}

	rows, err := s.db.Query(sql, args...)
	if err != nil {
		return nil, fmt.Errorf("%w: running search: %v", ErrDB, err)
	}
	defer rows.Close()

	out := []SearchHit{}
	for rows.Next() {
		var h SearchHit
		if err := rows.Scan(&h.Kind, &h.ID, &h.Title, &h.Snippet, &h.Score); err != nil {
			return nil, fmt.Errorf("%w: reading search hit: %v", ErrDB, err)
		}
		out = append(out, h)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("%w: running search: %v", ErrDB, err)
	}
	return out, nil
}

// ftsQuery turns free text into a safe FTS5 MATCH expression. Every term
// is quoted so characters FTS5 treats as operators cannot produce a syntax
// error from a user's plain words.
func ftsQuery(query string) string {
	fields := strings.Fields(query)
	quoted := make([]string, 0, len(fields))
	for _, field := range fields {
		cleaned := strings.ReplaceAll(field, `"`, "")
		if cleaned == "" {
			continue
		}
		quoted = append(quoted, `"`+cleaned+`"`)
	}
	if len(quoted) == 0 {
		// Every character was stripped; match a token that cannot occur.
		return `"__mk_no_match__"`
	}
	return strings.Join(quoted, " ")
}
