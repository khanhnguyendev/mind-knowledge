package render

import (
	"bytes"
	"strings"
	"testing"
)

func TestBoardGroupsStoriesUnderEpics(t *testing.T) {
	var buf bytes.Buffer
	Board(&buf, []BoardProject{
		{
			Name: "my-app",
			Epics: []BoardEpic{
				{
					ID:    "a3f2k1",
					Title: "Auth overhaul",
					Stories: []BoardStory{
						{ID: "b4g3l2", Status: "ready", Title: "add login endpoint"},
						{ID: "c5h4m3", Status: "review", Title: "migrate hashes"},
					},
				},
			},
		},
	})

	out := buf.String()
	for _, want := range []string{
		"my-app", "Auth overhaul", "a3f2k1",
		"b4g3l2", "add login endpoint", "ready",
		"c5h4m3", "migrate hashes", "review",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("board output missing %q:\n%s", want, out)
		}
	}

	// The epic heading must precede its stories.
	if strings.Index(out, "Auth overhaul") > strings.Index(out, "add login endpoint") {
		t.Errorf("epic heading printed after its stories:\n%s", out)
	}
}

func TestBoardEmptyProjectStillPrintsName(t *testing.T) {
	var buf bytes.Buffer
	Board(&buf, []BoardProject{{Name: "empty-proj"}})

	if !strings.Contains(buf.String(), "empty-proj") {
		t.Errorf("empty project omitted:\n%s", buf.String())
	}
}

func TestBoardNoProjectsPrintsNotice(t *testing.T) {
	var buf bytes.Buffer
	Board(&buf, nil)

	if strings.TrimSpace(buf.String()) == "" {
		t.Error("empty board printed nothing; want a human-readable notice")
	}
}
