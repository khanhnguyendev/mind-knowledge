package render

import (
	"bytes"
	"strings"
	"testing"

	"github.com/khanhnguyendev/mind-knowledge/internal/model"
)

func TestWikiIndexGroupsByKind(t *testing.T) {
	var buf bytes.Buffer
	WikiIndex(&buf, []model.WikiPage{
		{Slug: "auth-model", Title: "Auth Model", Kind: "concept",
			Summary: "How auth works", Status: "current"},
		{Slug: "specs/mk", Title: "mk spec", Kind: "spec",
			Summary: "The binary design", Status: "current"},
		{Slug: "old-idea", Title: "Old Idea", Kind: "concept",
			Summary: "Replaced", Status: "superseded"},
	})

	out := buf.String()

	for _, want := range []string{
		"# Wiki Index", "## concept", "## spec",
		"auth-model", "How auth works", "specs/mk", "The binary design",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("index missing %q:\n%s", want, out)
		}
	}

	if strings.Index(out, "## concept") > strings.Index(out, "## spec") {
		t.Errorf("kinds not alphabetical:\n%s", out)
	}
}

func TestWikiIndexMarksNonCurrentStatus(t *testing.T) {
	var buf bytes.Buffer
	WikiIndex(&buf, []model.WikiPage{
		{Slug: "old-idea", Title: "Old Idea", Kind: "concept",
			Summary: "Replaced", Status: "superseded"},
	})

	if !strings.Contains(buf.String(), "superseded") {
		t.Errorf("non-current status not shown:\n%s", buf.String())
	}
}

func TestWikiIndexHandlesMissingSummary(t *testing.T) {
	var buf bytes.Buffer
	WikiIndex(&buf, []model.WikiPage{
		{Slug: "bare", Title: "Bare Page", Kind: "note-less", Status: "current"},
	})

	out := buf.String()
	if !strings.Contains(out, "bare") {
		t.Errorf("page omitted when summary is empty:\n%s", out)
	}
	if strings.Contains(out, "—  —") {
		t.Errorf("empty summary rendered as stray punctuation:\n%s", out)
	}
}

func TestWikiIndexEmptyPrintsHeading(t *testing.T) {
	var buf bytes.Buffer
	WikiIndex(&buf, nil)

	if !strings.Contains(buf.String(), "# Wiki Index") {
		t.Errorf("empty index missing heading:\n%s", buf.String())
	}
}
