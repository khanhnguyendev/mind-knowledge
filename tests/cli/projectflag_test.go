package cli_test

import (
	"strings"
	"testing"
)

// This file pins the -p/--project contract end to end. Every command
// either scopes by it or rejects it; none accept it and quietly ignore it,
// because a skill threading -p through every call would then act on
// another project's rows believing it had scoped them.

// --- doctor honours -p ---

// twoProjects seeds two projects, each with an epic, a done-but-planless
// story, and a wiki page, and returns their ids along with the ids of the
// first project's story and page.
func twoProjects(t *testing.T, db string) (mineID, theirsID, myStory, myPage, theirStory, theirPage string) {
	t.Helper()

	seed := func(name string) (pid, story, page string) {
		pid = strings.TrimSpace(
			mk(t, db, "project", "add", "--name", name, "--path", t.TempDir()).stdout)
		eid := strings.TrimSpace(
			mk(t, db, "epic", "create", "--project", pid, "--title", name+" epic").stdout)
		story = strings.TrimSpace(
			mk(t, db, "story", "create", "--epic", eid, "--title", name+" story").stdout)
		requireCode(t, mk(t, db, "story", "mv", story, "--to", "done"), 0)
		page = strings.TrimSpace(
			mk(t, db, "wiki", "add", "--project", pid, "--title", name+" page").stdout)
		return pid, story, page
	}

	mineID, myStory, myPage = seed("mine")
	theirsID, theirStory, theirPage = seed("theirs")
	return
}

func doctorIDs(t *testing.T, db string, args ...string) map[string]bool {
	t.Helper()
	r := mk(t, db, append([]string{"--json", "doctor"}, args...)...)
	requireCode(t, r, 0)

	var findings []struct {
		Check string `json:"check"`
		ID    string `json:"id"`
	}
	decode(t, r, &findings)

	ids := map[string]bool{}
	for _, f := range findings {
		ids[f.ID] = true
	}
	return ids
}

func TestDoctorScopesFindingsToTheNamedProject(t *testing.T) {
	db := newDB(t)
	mineID, _, myStory, myPage, theirStory, theirPage := twoProjects(t, db)

	all := doctorIDs(t, db)
	for _, id := range []string{myStory, myPage, theirStory, theirPage} {
		if !all[id] {
			t.Fatalf("unscoped doctor did not report %s: %v", id, all)
		}
	}

	scoped := doctorIDs(t, db, "-p", mineID)
	if !scoped[myStory] || !scoped[myPage] {
		t.Errorf("doctor -p dropped the named project's own findings: %v", scoped)
	}
	// This is the whole point: before the fix, `mk doctor -p mine`
	// reported drift for every project on the machine.
	if scoped[theirStory] || scoped[theirPage] {
		t.Errorf("doctor -p reported another project's findings: %v", scoped)
	}
}

func TestDoctorAcceptsAProjectName(t *testing.T) {
	db := newDB(t)
	_, _, myStory, _, theirStory, _ := twoProjects(t, db)

	scoped := doctorIDs(t, db, "-p", "mine")
	if !scoped[myStory] || scoped[theirStory] {
		t.Errorf("doctor -p by name scoped wrongly: %v", scoped)
	}
}

func TestDoctorUnknownProjectExitsOne(t *testing.T) {
	requireCode(t, mk(t, newDB(t), "doctor", "-p", "no-such-project"), 1)
}

func TestDoctorUnknownScopeStillExitsTwo(t *testing.T) {
	// The scope error must keep its own class now that a bad -p is a
	// not-found rather than everything being flattened to bad input.
	requireCode(t, mk(t, newDB(t), "doctor", "--scope", "nonsense"), 2)
}

// --- the cross-project commands reject -p rather than ignoring it ---

func TestCrossProjectCommandsRejectTheProjectFlag(t *testing.T) {
	db := newDB(t)
	pid := seedProject(t, db)

	for _, args := range [][]string{
		{"search", "anything"},
		{"source", "ls"},
		{"source", "add", "--title", "t", "--body", "b"},
		{"tag", "ls"},
		{"tag", "add", "hot", "--on", "project:" + pid},
		{"link", "ls"},
	} {
		t.Run(strings.Join(args, " "), func(t *testing.T) {
			// A real project id: the rejection is about the flag having
			// no meaning here, not about the value being unresolvable.
			r := mk(t, db, append(args, "-p", pid)...)
			requireCode(t, r, 2)
			if !strings.Contains(r.stdout+r.stderr, "cross-project") {
				t.Errorf("output = %q, want it to explain why -p is refused", r.stdout+r.stderr)
			}
		})
	}
}

func TestCrossProjectCommandsStillWorkWithoutTheProjectFlag(t *testing.T) {
	db := newDB(t)
	pid := seedProject(t, db)

	requireCode(t, mk(t, db, "source", "add", "--title", "t", "--body", "b"), 0)
	requireCode(t, mk(t, db, "tag", "add", "hot", "--on", "project:"+pid), 0)
	for _, args := range [][]string{
		{"search", "t"}, {"source", "ls"}, {"tag", "ls"}, {"link", "ls"},
	} {
		requireCode(t, mk(t, db, args...), 0)
	}
}

// --- edit reassigns only through --set-project ---

func wikiProjectOf(t *testing.T, db, page string) string {
	t.Helper()
	r := mk(t, db, "--json", "wiki", "view", page)
	requireCode(t, r, 0)

	var got struct {
		ProjectID string `json:"project_id"`
	}
	decode(t, r, &got)
	return got.ProjectID
}

func epicProjectOf(t *testing.T, db, epic string) string {
	t.Helper()
	r := mk(t, db, "--json", "epic", "view", epic)
	requireCode(t, r, 0)

	var got struct {
		ProjectID string `json:"project_id"`
	}
	decode(t, r, &got)
	return got.ProjectID
}

func TestWikiEditDoesNotReassignThroughTheProjectFlag(t *testing.T) {
	db := newDB(t)
	pid := seedProject(t, db)

	// A deliberately cross-project page.
	page := strings.TrimSpace(mk(t, db, "wiki", "add", "--title", "Shared").stdout)

	requireCode(t, mk(t, db, "wiki", "edit", page, "-p", pid, "--summary", "touched"), 0)
	if got := wikiProjectOf(t, db, page); got != "" {
		t.Errorf("page was reassigned to %q by a plain -p edit; it must stay cross-project", got)
	}
}

func TestWikiEditReassignsThroughSetProject(t *testing.T) {
	db := newDB(t)
	pid := seedProject(t, db)
	page := strings.TrimSpace(mk(t, db, "wiki", "add", "--title", "Shared").stdout)

	requireCode(t, mk(t, db, "wiki", "edit", page, "--set-project", pid), 0)
	if got := wikiProjectOf(t, db, page); got != pid {
		t.Errorf("project after --set-project = %q, want %q", got, pid)
	}

	// An empty --set-project makes it cross-project again.
	requireCode(t, mk(t, db, "wiki", "edit", page, "--set-project", ""), 0)
	if got := wikiProjectOf(t, db, page); got != "" {
		t.Errorf("project after an empty --set-project = %q, want it cleared", got)
	}
}

func TestEpicEditDoesNotReassignThroughTheProjectFlag(t *testing.T) {
	db := newDB(t)
	pid := seedProject(t, db)
	other := strings.TrimSpace(
		mk(t, db, "project", "add", "--name", "other", "--path", t.TempDir()).stdout)
	eid := strings.TrimSpace(
		mk(t, db, "epic", "create", "--project", pid, "--title", "Auth").stdout)

	requireCode(t, mk(t, db, "epic", "edit", eid, "-p", other, "--title", "Auth v2"), 0)
	if got := epicProjectOf(t, db, eid); got != pid {
		t.Errorf("epic moved to %q by a plain -p edit; it must stay in %q", got, pid)
	}
}

func TestEpicEditReassignsThroughSetProject(t *testing.T) {
	db := newDB(t)
	pid := seedProject(t, db)
	other := strings.TrimSpace(
		mk(t, db, "project", "add", "--name", "other", "--path", t.TempDir()).stdout)
	eid := strings.TrimSpace(
		mk(t, db, "epic", "create", "--project", pid, "--title", "Auth").stdout)

	requireCode(t, mk(t, db, "epic", "edit", eid, "--set-project", other), 0)
	if got := epicProjectOf(t, db, eid); got != other {
		t.Errorf("project after --set-project = %q, want %q", got, other)
	}
}
