package model

import (
	"encoding/json"
	"testing"
)

func TestValidators(t *testing.T) {
	cases := []struct {
		name  string
		fn    func(string) bool
		good  []string
		bad   []string
	}{
		{"project status", ValidProjectStatus,
			[]string{"active", "paused", "archived"},
			[]string{"", "done", "Active"}},
		{"epic status", ValidEpicStatus,
			[]string{"backlog", "in-progress", "done", "dropped"},
			[]string{"", "ready", "review"}},
		{"story status", ValidStoryStatus,
			[]string{"backlog", "ready", "in-progress", "review", "done", "dropped"},
			[]string{"", "todo", "closed"}},
		{"priority", ValidPriority,
			[]string{"low", "med", "high"},
			[]string{"", "medium", "urgent"}},
		{"source kind", ValidSourceKind,
			[]string{"article", "paper", "transcript", "chapter", "asset", "note"},
			[]string{"", "spec", "video"}},
		{"wiki kind", ValidWikiKind,
			[]string{"summary", "concept", "entity", "decision", "spec", "synthesis", "comparison"},
			[]string{"", "source", "article"}},
		{"wiki status", ValidWikiStatus,
			[]string{"current", "stale", "superseded"},
			[]string{"", "draft", "active"}},
		{"relation", ValidRelation,
			[]string{"derived-from", "supersedes", "references", "implements"},
			[]string{"", "derived_from", "cites"}},
		{"entity kind", ValidEntityKind,
			[]string{"project", "epic", "story", "source", "wiki"},
			[]string{"", "page", "task"}},
	}

	for _, c := range cases {
		for _, v := range c.good {
			if !c.fn(v) {
				t.Errorf("%s: %q rejected, want accepted", c.name, v)
			}
		}
		for _, v := range c.bad {
			if c.fn(v) {
				t.Errorf("%s: %q accepted, want rejected", c.name, v)
			}
		}
	}
}

func TestSlugify(t *testing.T) {
	cases := map[string]string{
		"Auth Model":            "auth-model",
		"concepts/Auth Model":   "concepts/auth-model",
		"  Spaced  Out  ":       "spaced-out",
		"Punctuation! Here?":    "punctuation-here",
		"Already-slugged":       "already-slugged",
		"Multiple///Slashes":    "multiple/slashes",
		"/Auth Model":           "auth-model",        // regression: leading slash trimmed
		"Auth Model/":           "auth-model",        // regression: trailing slash trimmed
		"///":                   "",                  // regression: all slashes reduce to empty
	}
	for in, want := range cases {
		if got := Slugify(in); got != want {
			t.Errorf("Slugify(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestStoryJSONFieldNames(t *testing.T) {
	b, err := json.Marshal(Story{ID: "a3f2k1", EpicID: "b4g3l2", Title: "t", Status: "ready"})
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	var got map[string]any
	if err := json.Unmarshal(b, &got); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	for _, key := range []string{"id", "epic_id", "title", "status", "priority", "position"} {
		if _, ok := got[key]; !ok {
			t.Errorf("Story JSON missing key %q; got %v", key, got)
		}
	}
	// Empty optional fields must be omitted so skills can test presence.
	if _, ok := got["plan"]; ok {
		t.Errorf("Story JSON should omit empty plan; got %v", got)
	}
}
