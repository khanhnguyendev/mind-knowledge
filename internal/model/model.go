// Package model defines mk's entities and the closed sets of values their
// enumerated fields accept. It has no dependencies on storage or the CLI.
package model

import "strings"

type Project struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	RepoPath  string `json:"repo_path"`
	GitRemote string `json:"git_remote,omitempty"`
	Status    string `json:"status"`
	CreatedAt string `json:"created_at"`
	UpdatedAt string `json:"updated_at"`
}

type Epic struct {
	ID          string `json:"id"`
	ProjectID   string `json:"project_id"`
	Title       string `json:"title"`
	Description string `json:"description,omitempty"`
	Status      string `json:"status"`
	Position    int    `json:"position"`
	CreatedAt   string `json:"created_at"`
	UpdatedAt   string `json:"updated_at"`
}

type Story struct {
	ID          string `json:"id"`
	EpicID      string `json:"epic_id"`
	Title       string `json:"title"`
	Description string `json:"description,omitempty"`
	Status      string `json:"status"`
	Priority    string `json:"priority"`
	Position    int    `json:"position"`
	Acceptance  string `json:"acceptance,omitempty"`
	Plan        string `json:"plan,omitempty"`
	Notes       string `json:"notes,omitempty"`
	CreatedAt   string `json:"created_at"`
	UpdatedAt   string `json:"updated_at"`
}

type Source struct {
	ID          string `json:"id"`
	URI         string `json:"uri,omitempty"`
	Title       string `json:"title"`
	Kind        string `json:"kind"`
	Body        string `json:"body,omitempty"`
	AssetPath   string `json:"asset_path,omitempty"`
	ContentHash string `json:"content_hash"`
	IngestedAt  string `json:"ingested_at"`
}

type WikiPage struct {
	ID        string `json:"id"`
	Slug      string `json:"slug"`
	Title     string `json:"title"`
	Summary   string `json:"summary,omitempty"`
	Kind      string `json:"kind"`
	Body      string `json:"body,omitempty"`
	Status    string `json:"status"`
	ProjectID string `json:"project_id,omitempty"`
	CreatedAt string `json:"created_at"`
	UpdatedAt string `json:"updated_at"`
}

type Link struct {
	FromKind string `json:"from_kind"`
	FromID   string `json:"from_id"`
	ToKind   string `json:"to_kind"`
	ToID     string `json:"to_id"`
	Relation string `json:"relation"`
}

type LogEntry struct {
	ID        int64  `json:"id"`
	TS        string `json:"ts"`
	Kind      string `json:"kind"`
	ProjectID string `json:"project_id,omitempty"`
	Ref       string `json:"ref,omitempty"`
	Summary   string `json:"summary"`
}

// oneOf reports whether v appears in allowed.
func oneOf(v string, allowed ...string) bool {
	for _, a := range allowed {
		if v == a {
			return true
		}
	}
	return false
}

func ValidProjectStatus(v string) bool {
	return oneOf(v, "active", "paused", "archived")
}

func ValidEpicStatus(v string) bool {
	return oneOf(v, "backlog", "in-progress", "done", "dropped")
}

func ValidStoryStatus(v string) bool {
	return oneOf(v, "backlog", "ready", "in-progress", "review", "done", "dropped")
}

func ValidPriority(v string) bool {
	return oneOf(v, "low", "med", "high")
}

func ValidSourceKind(v string) bool {
	return oneOf(v, "article", "paper", "transcript", "chapter", "asset", "note")
}

func ValidWikiKind(v string) bool {
	return oneOf(v, "summary", "concept", "entity", "decision", "spec", "synthesis", "comparison")
}

func ValidWikiStatus(v string) bool {
	return oneOf(v, "current", "stale", "superseded")
}

func ValidRelation(v string) bool {
	return oneOf(v, "derived-from", "supersedes", "references", "implements")
}

func ValidEntityKind(v string) bool {
	return oneOf(v, "project", "epic", "story", "source", "wiki")
}

// Slugify converts a title into a wiki slug: lowercase, alphanumerics and
// hyphens only, with '/' preserved as a path separator.
func Slugify(s string) string {
	var b strings.Builder
	lastRune := rune(0)
	for _, r := range strings.ToLower(strings.TrimSpace(s)) {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9':
			b.WriteRune(r)
			lastRune = r
		case r == '/':
			// Only write slash if last rune wasn't a slash
			if lastRune != '/' {
				trimTrailingHyphen(&b)
				b.WriteRune('/')
				lastRune = '/'
			}
		default:
			// Only write hyphen if last rune wasn't a hyphen or slash
			if lastRune != '-' && lastRune != '/' {
				b.WriteRune('-')
				lastRune = '-'
			}
		}
	}
	out := b.String()
	return strings.Trim(out, "-/")
}

// trimTrailingHyphen removes a hyphen immediately before a path separator
// so "concepts- /x" cannot occur.
func trimTrailingHyphen(b *strings.Builder) {
	s := b.String()
	if strings.HasSuffix(s, "-") {
		b.Reset()
		b.WriteString(strings.TrimSuffix(s, "-"))
	}
}
