package cli_test

import (
	"strings"
	"testing"
)

func TestTagAddAndListForEntity(t *testing.T) {
	db := newDB(t)
	eid := seedEpic(t, db)
	sid := strings.TrimSpace(mk(t, db, "story", "create", "--epic", eid, "--title", "x").stdout)

	requireCode(t, mk(t, db, "tag", "add", "oauth", "--on", "story:"+sid), 0)

	r := mk(t, db, "--json", "tag", "ls", "--on", "story:"+sid)
	requireCode(t, r, 0)

	var tags []string
	decode(t, r, &tags)
	if len(tags) != 1 || tags[0] != "oauth" {
		t.Errorf("tags = %v, want [oauth]", tags)
	}
}

func TestTagListAllNames(t *testing.T) {
	db := newDB(t)
	eid := seedEpic(t, db)
	sid := strings.TrimSpace(mk(t, db, "story", "create", "--epic", eid, "--title", "x").stdout)

	mk(t, db, "tag", "add", "zeta", "--on", "story:"+sid)
	mk(t, db, "tag", "add", "alpha", "--on", "story:"+sid)

	r := mk(t, db, "--json", "tag", "ls")
	requireCode(t, r, 0)

	var tags []string
	decode(t, r, &tags)
	if len(tags) != 2 || tags[0] != "alpha" {
		t.Errorf("tags = %v, want [alpha zeta]", tags)
	}
}

func TestTagListEntitiesWithTag(t *testing.T) {
	db := newDB(t)
	eid := seedEpic(t, db)
	sid := strings.TrimSpace(mk(t, db, "story", "create", "--epic", eid, "--title", "x").stdout)

	mk(t, db, "tag", "add", "oauth", "--on", "story:"+sid)
	mk(t, db, "tag", "add", "oauth", "--on", "epic:"+eid)

	r := mk(t, db, "--json", "tag", "ls", "--tag", "oauth")
	requireCode(t, r, 0)

	var tagged []struct {
		FromKind string `json:"from_kind"`
	}
	decode(t, r, &tagged)
	if len(tagged) != 2 {
		t.Errorf("tagged = %+v, want 2", tagged)
	}
}

func TestTagAddMissingEntityExitsOne(t *testing.T) {
	db := newDB(t)

	r := mk(t, db, "tag", "add", "oauth", "--on", "story:nope99")
	requireCode(t, r, 1)
}

func TestTagAddMalformedTargetExitsTwo(t *testing.T) {
	db := newDB(t)

	r := mk(t, db, "tag", "add", "oauth", "--on", "nope99")
	requireCode(t, r, 2)
}

func TestTagRemove(t *testing.T) {
	db := newDB(t)
	eid := seedEpic(t, db)
	sid := strings.TrimSpace(mk(t, db, "story", "create", "--epic", eid, "--title", "x").stdout)

	mk(t, db, "tag", "add", "oauth", "--on", "story:"+sid)
	requireCode(t, mk(t, db, "tag", "rm", "oauth", "--on", "story:"+sid), 0)
	requireCode(t, mk(t, db, "tag", "rm", "oauth", "--on", "story:"+sid), 1)
}

func TestTagNormalizesCase(t *testing.T) {
	db := newDB(t)
	eid := seedEpic(t, db)
	sid := strings.TrimSpace(mk(t, db, "story", "create", "--epic", eid, "--title", "x").stdout)

	// Add with mixed case
	requireCode(t, mk(t, db, "tag", "add", "OAuth", "--on", "story:"+sid), 0)

	// Query with different case should work (and find the normalized form)
	r := mk(t, db, "--json", "tag", "ls", "--on", "story:"+sid)
	requireCode(t, r, 0)

	var tags []string
	decode(t, r, &tags)
	if len(tags) != 1 || tags[0] != "oauth" {
		t.Errorf("tags = %v, want [oauth] (normalized)", tags)
	}

	// Adding with different case is idempotent
	requireCode(t, mk(t, db, "tag", "add", "OAUTH", "--on", "story:"+sid), 0)
	r = mk(t, db, "--json", "tag", "ls", "--on", "story:"+sid)
	decode(t, r, &tags)
	if len(tags) != 1 {
		t.Errorf("after variant add: tags = %v, want still 1", tags)
	}
}

func TestTagRemoveRejectsMalformedTarget(t *testing.T) {
	db := newDB(t)

	// Malformed target should exit 2
	r := mk(t, db, "tag", "rm", "oauth", "--on", "nope99")
	requireCode(t, r, 2)
}

func TestTagRemoveMissingEntityExitsOne(t *testing.T) {
	db := newDB(t)

	// Well-formed target naming absent entity should exit 1
	r := mk(t, db, "tag", "rm", "oauth", "--on", "story:nope99")
	requireCode(t, r, 1)
}

func TestTagRemoveEmptyNameExitsTwo(t *testing.T) {
	db := newDB(t)
	eid := seedEpic(t, db)
	sid := strings.TrimSpace(mk(t, db, "story", "create", "--epic", eid, "--title", "x").stdout)

	// RemoveTag with empty name should exit 2 (bad input), same as AddTag
	r := mk(t, db, "tag", "rm", "", "--on", "story:"+sid)
	requireCode(t, r, 2)
}

func TestTagEmptyListAllNames(t *testing.T) {
	db := newDB(t)

	// List tags when none exist should return empty JSON array
	r := mk(t, db, "--json", "tag", "ls")
	requireCode(t, r, 0)

	// Verify output is "[]" not "null"
	if r.stdout != "[]\n" {
		t.Errorf("empty tag list stdout = %q, want []\\n", r.stdout)
	}

	var tags []string
	decode(t, r, &tags)
	if tags == nil || len(tags) != 0 {
		t.Errorf("decoded tags = %v, want empty slice", tags)
	}
}

func TestTagEmptyListForEntity(t *testing.T) {
	db := newDB(t)
	eid := seedEpic(t, db)
	sid := strings.TrimSpace(mk(t, db, "story", "create", "--epic", eid, "--title", "x").stdout)

	// List tags for entity with no tags should return empty JSON array
	r := mk(t, db, "--json", "tag", "ls", "--on", "story:"+sid)
	requireCode(t, r, 0)

	// Verify output is "[]" not "null"
	if r.stdout != "[]\n" {
		t.Errorf("empty tags for entity stdout = %q, want []\\n", r.stdout)
	}

	var tags []string
	decode(t, r, &tags)
	if tags == nil || len(tags) != 0 {
		t.Errorf("decoded tags = %v, want empty slice", tags)
	}
}

func TestTagEmptyListEntitiesWithTag(t *testing.T) {
	db := newDB(t)

	// List entities with nonexistent tag should return empty JSON array
	r := mk(t, db, "--json", "tag", "ls", "--tag", "nonexistent")
	requireCode(t, r, 0)

	// Verify output is "[]" not "null"
	if r.stdout != "[]\n" {
		t.Errorf("empty entities for tag stdout = %q, want []\\n", r.stdout)
	}

	var tagged []interface{}
	decode(t, r, &tagged)
	if tagged == nil || len(tagged) != 0 {
		t.Errorf("decoded tagged = %v, want empty slice", tagged)
	}
}
