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
