package cli_test

import (
	"strings"
	"testing"
)

// This file reproduces, against the binary, the two ways a deleted entity
// used to leave rows behind. links and entity_tags carry no foreign key to
// the entity they name — their endpoints span five tables, so no single
// REFERENCES clause could express it — so cleanup is Delete*'s job.
//
// The cost of skipping it is not cosmetic. mk is a deliberate dumb core
// that never enforces workflow, which makes doctor the only thing that
// catches the resulting drift; a leaked edge permanently vouches for
// whichever endpoint survived, and silently switches three doctor checks
// off for it.

func doctorChecks(t *testing.T, db string, scope string) []struct {
	Check string `json:"check"`
	ID    string `json:"id"`
} {
	t.Helper()
	r := mk(t, db, "--json", "doctor", "--scope", scope)
	requireCode(t, r, 0)

	var findings []struct {
		Check string `json:"check"`
		ID    string `json:"id"`
	}
	decode(t, r, &findings)
	return findings
}

func hasFinding(t *testing.T, db, scope, check, id string) bool {
	t.Helper()
	for _, f := range doctorChecks(t, db, scope) {
		if f.Check == check && (id == "" || f.ID == id) {
			return true
		}
	}
	return false
}

func TestDeletingADerivedPageMakesItsSourceUnprocessedAgain(t *testing.T) {
	db := newDB(t)

	sid := strings.TrimSpace(
		mk(t, db, "source", "add", "--title", "An article", "--body", "text").stdout)
	page := strings.TrimSpace(
		mk(t, db, "wiki", "add", "--title", "Derived", "--body", "b").stdout)
	requireCode(t, mk(t, db, "link", "add",
		"--from", "wiki:"+page, "--to", "source:"+sid, "--relation", "derived-from"), 0)

	if hasFinding(t, db, "wiki", "wiki.unprocessed", sid) {
		t.Fatalf("source reported unprocessed while a page still derives from it")
	}

	requireCode(t, mk(t, db, "wiki", "rm", page), 0)

	// The page that vouched for this source is gone. Before the fix its
	// edge survived and kept vouching, so doctor never named the source
	// again — a wiki that is entirely unprocessed got a clean bill of
	// health.
	if !hasFinding(t, db, "wiki", "wiki.unprocessed", sid) {
		t.Errorf("source not reported unprocessed after its only derived page was deleted")
	}
	if hasFinding(t, db, "wiki", "wiki.dangling", "") {
		t.Errorf("wiki rm left a dangling edge behind")
	}
}

func TestDeletingALinkingPageMakesItsTargetAnOrphanAgain(t *testing.T) {
	db := newDB(t)

	hub := strings.TrimSpace(mk(t, db, "wiki", "add", "--title", "Hub", "--body", "b").stdout)
	target := strings.TrimSpace(mk(t, db, "wiki", "add", "--title", "Target", "--body", "b").stdout)
	requireCode(t, mk(t, db, "link", "add",
		"--from", "wiki:"+hub, "--to", "wiki:"+target, "--relation", "references"), 0)

	if hasFinding(t, db, "wiki", "wiki.orphans", target) {
		t.Fatalf("target reported orphan while an inbound link still exists")
	}

	requireCode(t, mk(t, db, "wiki", "rm", hub), 0)

	if !hasFinding(t, db, "wiki", "wiki.orphans", target) {
		t.Errorf("target not reported orphan after its only inbound link was deleted")
	}
}

func TestDeletedStoryDisappearsFromItsTags(t *testing.T) {
	db := newDB(t)

	eid := seedEpic(t, db)
	sid := strings.TrimSpace(
		mk(t, db, "story", "create", "--epic", eid, "--title", "tagged").stdout)
	requireCode(t, mk(t, db, "tag", "add", "hot", "--on", "story:"+sid), 0)

	requireCode(t, mk(t, db, "story", "rm", sid), 0)

	// A skill iterating a tag must not be handed an id that story view
	// then rejects with exit 1.
	r := mk(t, db, "--json", "tag", "ls", "--tag", "hot")
	requireCode(t, r, 0)
	if strings.TrimSpace(r.stdout) != "[]" {
		t.Errorf("tag ls --tag hot = %s, want [] after the story was deleted", r.stdout)
	}
	if strings.Contains(r.stdout, sid) {
		t.Errorf("tag ls still returns the deleted story %s: %s", sid, r.stdout)
	}
}

func TestDeletedSourceDisappearsFromItsLinksAndTags(t *testing.T) {
	db := newDB(t)

	sid := strings.TrimSpace(
		mk(t, db, "source", "add", "--title", "An article", "--body", "text").stdout)
	page := strings.TrimSpace(
		mk(t, db, "wiki", "add", "--title", "Derived", "--body", "b").stdout)
	requireCode(t, mk(t, db, "link", "add",
		"--from", "wiki:"+page, "--to", "source:"+sid, "--relation", "derived-from"), 0)
	requireCode(t, mk(t, db, "tag", "add", "hot", "--on", "source:"+sid), 0)

	requireCode(t, mk(t, db, "source", "rm", sid), 0)

	links := mk(t, db, "--json", "link", "ls")
	requireCode(t, links, 0)
	if strings.TrimSpace(links.stdout) != "[]" {
		t.Errorf("link ls = %s, want [] after the source was deleted", links.stdout)
	}

	tags := mk(t, db, "--json", "tag", "ls", "--tag", "hot")
	requireCode(t, tags, 0)
	if strings.TrimSpace(tags.stdout) != "[]" {
		t.Errorf("tag ls --tag hot = %s, want [] after the source was deleted", tags.stdout)
	}

	// The page that cited it is uncited again, not still vouched for.
	if !hasFinding(t, db, "wiki", "wiki.uncited", page) {
		t.Errorf("page not reported uncited after the source it cited was deleted")
	}
}

// TestDeletingAProjectCleansCascadedChildren covers the level the leak
// reappears at if only the named row is cleaned: the database cascades
// projects -> epics -> stories, so those children vanish without any
// Delete* running for them.
func TestDeletingAProjectCleansCascadedChildren(t *testing.T) {
	db := newDB(t)

	pid := seedProject(t, db)
	eid := strings.TrimSpace(
		mk(t, db, "epic", "create", "--project", pid, "--title", "Auth").stdout)
	sid := strings.TrimSpace(
		mk(t, db, "story", "create", "--epic", eid, "--title", "login").stdout)
	page := strings.TrimSpace(
		mk(t, db, "wiki", "add", "--title", "Auth Model", "--body", "b").stdout)

	requireCode(t, mk(t, db, "link", "add",
		"--from", "story:"+sid, "--to", "wiki:"+page, "--relation", "implements"), 0)
	requireCode(t, mk(t, db, "link", "add",
		"--from", "epic:"+eid, "--to", "wiki:"+page, "--relation", "references"), 0)
	requireCode(t, mk(t, db, "tag", "add", "hot", "--on", "story:"+sid), 0)
	requireCode(t, mk(t, db, "tag", "add", "hot", "--on", "epic:"+eid), 0)
	requireCode(t, mk(t, db, "tag", "add", "hot", "--on", "project:"+pid), 0)

	requireCode(t, mk(t, db, "project", "rm", pid), 0)

	links := mk(t, db, "--json", "link", "ls")
	requireCode(t, links, 0)
	if strings.TrimSpace(links.stdout) != "[]" {
		t.Errorf("link ls = %s, want [] after the project cascade", links.stdout)
	}

	tags := mk(t, db, "--json", "tag", "ls", "--tag", "hot")
	requireCode(t, tags, 0)
	if strings.TrimSpace(tags.stdout) != "[]" {
		t.Errorf("tag ls --tag hot = %s, want [] after the project cascade", tags.stdout)
	}

	if hasFinding(t, db, "wiki", "wiki.dangling", "") {
		t.Errorf("project rm left dangling edges from its cascaded children")
	}
	if !hasFinding(t, db, "wiki", "wiki.orphans", page) {
		t.Errorf("page not reported orphan after every edge into it was deleted")
	}
}

// TestDeletingAnEpicCleansItsCascadedStories is the same leak one level
// down: epics -> stories also cascades in the database.
func TestDeletingAnEpicCleansItsCascadedStories(t *testing.T) {
	db := newDB(t)

	eid := seedEpic(t, db)
	sid := strings.TrimSpace(
		mk(t, db, "story", "create", "--epic", eid, "--title", "login").stdout)
	page := strings.TrimSpace(
		mk(t, db, "wiki", "add", "--title", "Auth Model", "--body", "b").stdout)

	requireCode(t, mk(t, db, "link", "add",
		"--from", "story:"+sid, "--to", "wiki:"+page, "--relation", "implements"), 0)
	requireCode(t, mk(t, db, "tag", "add", "hot", "--on", "story:"+sid), 0)

	requireCode(t, mk(t, db, "epic", "rm", eid), 0)

	links := mk(t, db, "--json", "link", "ls")
	requireCode(t, links, 0)
	if strings.TrimSpace(links.stdout) != "[]" {
		t.Errorf("link ls = %s, want [] after the epic cascade", links.stdout)
	}

	tags := mk(t, db, "--json", "tag", "ls", "--tag", "hot")
	requireCode(t, tags, 0)
	if strings.TrimSpace(tags.stdout) != "[]" {
		t.Errorf("tag ls --tag hot = %s, want [] after the epic cascade", tags.stdout)
	}
}

// TestDeleteLeavesUnrelatedEdgesAlone is the negative half: cleanup must
// remove the deleted entity's own rows and nothing else.
func TestDeleteLeavesUnrelatedEdgesAlone(t *testing.T) {
	db := newDB(t)

	sid := strings.TrimSpace(
		mk(t, db, "source", "add", "--title", "An article", "--body", "text").stdout)
	doomed := strings.TrimSpace(mk(t, db, "wiki", "add", "--title", "Doomed", "--body", "b").stdout)
	keeper := strings.TrimSpace(mk(t, db, "wiki", "add", "--title", "Keeper", "--body", "b").stdout)

	requireCode(t, mk(t, db, "link", "add",
		"--from", "wiki:"+doomed, "--to", "source:"+sid, "--relation", "derived-from"), 0)
	requireCode(t, mk(t, db, "link", "add",
		"--from", "wiki:"+keeper, "--to", "source:"+sid, "--relation", "derived-from"), 0)
	requireCode(t, mk(t, db, "tag", "add", "hot", "--on", "wiki:"+keeper), 0)

	requireCode(t, mk(t, db, "wiki", "rm", doomed), 0)

	links := mk(t, db, "--json", "link", "ls")
	requireCode(t, links, 0)
	if strings.Count(links.stdout, "derived-from") != 1 {
		t.Errorf("link ls = %s, want exactly the surviving page's edge", links.stdout)
	}
	if !strings.Contains(links.stdout, keeper) {
		t.Errorf("the surviving page's edge was deleted too: %s", links.stdout)
	}

	tags := mk(t, db, "--json", "tag", "ls", "--tag", "hot")
	requireCode(t, tags, 0)
	if !strings.Contains(tags.stdout, keeper) {
		t.Errorf("the surviving page's tag was deleted too: %s", tags.stdout)
	}

	// The source is still cited by the surviving page.
	if hasFinding(t, db, "wiki", "wiki.unprocessed", sid) {
		t.Errorf("source wrongly reported unprocessed while a page still derives from it")
	}
}
