package cli_test

import (
	"strings"
	"testing"
)

func TestLinkAddCitation(t *testing.T) {
	db := newDB(t)

	src := strings.TrimSpace(
		mk(t, db, "source", "add", "--title", "An article", "--body", "text").stdout)
	mk(t, db, "wiki", "add", "--title", "Auth Model", "--kind", "summary", "--body", "b")

	r := mk(t, db, "--json", "link", "add",
		"--from", "wiki:auth-model", "--to", "source:"+src, "--relation", "derived-from")
	requireCode(t, r, 0)

	var l struct {
		FromKind string `json:"from_kind"`
		ToKind   string `json:"to_kind"`
		Relation string `json:"relation"`
	}
	decode(t, r, &l)
	if l.FromKind != "wiki" || l.ToKind != "source" || l.Relation != "derived-from" {
		t.Errorf("link = %+v", l)
	}
}

func TestLinkAddRejectsMalformedRef(t *testing.T) {
	db := newDB(t)

	r := mk(t, db, "link", "add", "--from", "auth-model", "--to", "source:x",
		"--relation", "derived-from")
	requireCode(t, r, 2)
}

func TestLinkAddRejectsUnknownRelation(t *testing.T) {
	db := newDB(t)

	mk(t, db, "wiki", "add", "--title", "A", "--body", "b")
	mk(t, db, "wiki", "add", "--title", "B", "--body", "b")

	r := mk(t, db, "link", "add", "--from", "wiki:a", "--to", "wiki:b", "--relation", "cites")
	requireCode(t, r, 2)
}

func TestLinkAddMissingEndpointExitsOne(t *testing.T) {
	db := newDB(t)

	mk(t, db, "wiki", "add", "--title", "A", "--body", "b")

	r := mk(t, db, "link", "add", "--from", "wiki:a", "--to", "source:nope99",
		"--relation", "derived-from")
	requireCode(t, r, 1)
}

func TestLinkListInbound(t *testing.T) {
	db := newDB(t)

	mk(t, db, "wiki", "add", "--title", "A", "--body", "b")
	mk(t, db, "wiki", "add", "--title", "B", "--body", "b")
	mk(t, db, "wiki", "add", "--title", "C", "--body", "b")

	mk(t, db, "link", "add", "--from", "wiki:a", "--to", "wiki:b", "--relation", "references")
	mk(t, db, "link", "add", "--from", "wiki:c", "--to", "wiki:b", "--relation", "supersedes")

	r := mk(t, db, "--json", "link", "ls", "--to", "wiki:b")
	requireCode(t, r, 0)

	var links []struct {
		Relation string `json:"relation"`
	}
	decode(t, r, &links)
	if len(links) != 2 {
		t.Errorf("inbound links = %+v, want 2", links)
	}
}

// TestLinkListEmptyIsJSONArray guards against the classic Go null-vs-[]
// bug: json.Unmarshal treats "null" and "[]" identically, so a test that
// decodes into a slice and checks len() == 0 would pass whether ls printed
// "[]" or "null". Asserting on the raw stdout text is the only way to
// actually pin the contract that empty result sets serialize as [].
func TestLinkListEmptyIsJSONArray(t *testing.T) {
	db := newDB(t)

	r := mk(t, db, "--json", "link", "ls")
	requireCode(t, r, 0)

	got := strings.TrimSpace(r.stdout)
	if got != "[]" {
		t.Errorf("stdout = %q, want the literal text \"[]\"", got)
	}
}

func TestLinkRemove(t *testing.T) {
	db := newDB(t)

	mk(t, db, "wiki", "add", "--title", "A", "--body", "b")
	mk(t, db, "wiki", "add", "--title", "B", "--body", "b")
	mk(t, db, "link", "add", "--from", "wiki:a", "--to", "wiki:b", "--relation", "references")

	requireCode(t, mk(t, db, "link", "rm",
		"--from", "wiki:a", "--to", "wiki:b", "--relation", "references"), 0)

	// Removing it a second time reports not found.
	requireCode(t, mk(t, db, "link", "rm",
		"--from", "wiki:a", "--to", "wiki:b", "--relation", "references"), 1)
}
