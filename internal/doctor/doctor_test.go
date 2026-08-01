package doctor

import (
	"database/sql"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	_ "modernc.org/sqlite"

	"github.com/khanhnguyendev/mind-knowledge/internal/store"
)

func testStore(t *testing.T) *store.Store {
	t.Helper()
	s, _ := testStoreAt(t)
	return s
}

// testStoreAt also hands back the database path, for the few tests that
// need to reach past the store and manipulate rows directly.
func testStoreAt(t *testing.T) (*store.Store, string) {
	t.Helper()
	path := filepath.Join(t.TempDir(), "test.db")
	s, err := store.Open(path)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	t.Cleanup(func() { s.Close() })
	return s, path
}

// forceDelete removes a row behind the store's back, reproducing the
// dangling edges that databases written by older binaries still carry.
func forceDelete(t *testing.T, path, table, id string) {
	t.Helper()
	db, err := sql.Open("sqlite", path)
	if err != nil {
		t.Fatalf("opening %s directly: %v", path, err)
	}
	defer db.Close()
	if _, err := db.Exec(`DELETE FROM `+table+` WHERE id = ?`, id); err != nil {
		t.Fatalf("deleting from %s: %v", table, err)
	}
}

// hasCheck reports whether findings contain a given check, optionally for a
// specific entity id.
func hasCheck(findings []Finding, check, id string) bool {
	for _, f := range findings {
		if f.Check == check && (id == "" || f.ID == id) {
			return true
		}
	}
	return false
}

func TestWikiOrphansFlagsPagesWithNoInboundLinks(t *testing.T) {
	s := testStore(t)

	orphan, _ := s.CreateWikiPage("", "Orphan", "concept", "", "b", "")
	target, _ := s.CreateWikiPage("", "Linked", "concept", "", "b", "")
	s.AddLink("wiki", orphan.ID, "wiki", target.ID, "references")

	findings, err := Run(s, []string{"wiki"}, "")
	if err != nil {
		t.Fatalf("Run: %v", err)
	}

	if !hasCheck(findings, "wiki.orphans", orphan.ID) {
		t.Errorf("orphan page not flagged: %+v", findings)
	}
	if hasCheck(findings, "wiki.orphans", target.ID) {
		t.Errorf("linked page wrongly flagged as an orphan: %+v", findings)
	}
}

func TestWikiStaleFlagsSupersededPagesStillMarkedCurrent(t *testing.T) {
	s := testStore(t)

	old, _ := s.CreateWikiPage("", "Old", "concept", "", "b", "")
	replacement, _ := s.CreateWikiPage("", "New", "concept", "", "b", "")
	s.AddLink("wiki", replacement.ID, "wiki", old.ID, "supersedes")

	findings, _ := Run(s, []string{"wiki"}, "")
	if !hasCheck(findings, "wiki.stale", old.ID) {
		t.Errorf("superseded page not flagged: %+v", findings)
	}
}

func TestWikiUncitedFlagsPagesWithNoDerivedFromEdge(t *testing.T) {
	s := testStore(t)

	src, _ := s.CreateSource("", "An article", "article", "text", "")
	cited, _ := s.CreateWikiPage("", "Cited", "summary", "", "b", "")
	uncited, _ := s.CreateWikiPage("", "Uncited", "summary", "", "b", "")
	s.AddLink("wiki", cited.ID, "source", src.ID, "derived-from")

	findings, _ := Run(s, []string{"wiki"}, "")
	if !hasCheck(findings, "wiki.uncited", uncited.ID) {
		t.Errorf("uncited page not flagged: %+v", findings)
	}
	if hasCheck(findings, "wiki.uncited", cited.ID) {
		t.Errorf("cited page wrongly flagged: %+v", findings)
	}
}

func TestWikiUnprocessedFlagsSourcesWithNoDerivedPage(t *testing.T) {
	s := testStore(t)

	used, _ := s.CreateSource("", "Used", "article", "text", "")
	unused, _ := s.CreateSource("", "Unused", "article", "other text", "")
	page, _ := s.CreateWikiPage("", "Summary", "summary", "", "b", "")
	s.AddLink("wiki", page.ID, "source", used.ID, "derived-from")

	findings, _ := Run(s, []string{"wiki"}, "")
	if !hasCheck(findings, "wiki.unprocessed", unused.ID) {
		t.Errorf("unprocessed source not flagged: %+v", findings)
	}
	if hasCheck(findings, "wiki.unprocessed", used.ID) {
		t.Errorf("processed source wrongly flagged: %+v", findings)
	}
}

func TestWikiMissingFlagsWikilinksWithNoPage(t *testing.T) {
	s := testStore(t)

	s.CreateWikiPage("", "Hub", "concept", "",
		"See [[auth-model]] and [[does-not-exist]].", "")
	s.CreateWikiPage("auth-model", "Auth Model", "concept", "", "b", "")

	findings, _ := Run(s, []string{"wiki"}, "")

	if !hasCheck(findings, "wiki.missing", "does-not-exist") {
		t.Errorf("missing wikilink target not flagged: %+v", findings)
	}
	if hasCheck(findings, "wiki.missing", "auth-model") {
		t.Errorf("resolvable wikilink wrongly flagged: %+v", findings)
	}
}

func TestStoryPlanlessFlagsDoneStoriesWithNoPlan(t *testing.T) {
	s := testStore(t)

	p, _ := s.CreateProject("my-app", "/tmp/my-app", "")
	e, _ := s.CreateEpic(p.ID, "Auth", "")
	bare, _ := s.CreateStory(e.ID, "no plan", "")
	planned, _ := s.CreateStory(e.ID, "has plan", "")

	done := "done"
	plan := "1. do the thing"
	s.UpdateStory(bare.ID, store.StoryFields{Status: &done})
	s.UpdateStory(planned.ID, store.StoryFields{Status: &done, Plan: &plan})

	findings, _ := Run(s, []string{"stories"}, "")
	if !hasCheck(findings, "story.planless", bare.ID) {
		t.Errorf("done story with no plan not flagged: %+v", findings)
	}
	if hasCheck(findings, "story.planless", planned.ID) {
		t.Errorf("planned story wrongly flagged: %+v", findings)
	}
}

func TestStoryStrandedFlagsLongRunningWork(t *testing.T) {
	s := testStore(t)

	p, _ := s.CreateProject("my-app", "/tmp/my-app", "")
	e, _ := s.CreateEpic(p.ID, "Auth", "")

	// Freeze the clock far enough back that the story looks stranded.
	original := store.Now
	store.Now = func() time.Time { return time.Now().UTC().AddDate(0, 0, -30) }
	old, _ := s.CreateStory(e.ID, "stuck", "")
	inProgress := "in-progress"
	s.UpdateStory(old.ID, store.StoryFields{Status: &inProgress})
	store.Now = original

	fresh, _ := s.CreateStory(e.ID, "just started", "")
	s.UpdateStory(fresh.ID, store.StoryFields{Status: &inProgress})

	findings, _ := Run(s, []string{"stories"}, "")
	if !hasCheck(findings, "story.stranded", old.ID) {
		t.Errorf("stranded story not flagged: %+v", findings)
	}
	if hasCheck(findings, "story.stranded", fresh.ID) {
		t.Errorf("fresh story wrongly flagged: %+v", findings)
	}
}

func TestEpicEmptyFlagsEpicsWithNoStories(t *testing.T) {
	s := testStore(t)

	p, _ := s.CreateProject("my-app", "/tmp/my-app", "")
	empty, _ := s.CreateEpic(p.ID, "Empty", "")
	filled, _ := s.CreateEpic(p.ID, "Filled", "")
	s.CreateStory(filled.ID, "a story", "")

	findings, _ := Run(s, []string{"stories"}, "")
	if !hasCheck(findings, "epic.empty", empty.ID) {
		t.Errorf("empty epic not flagged: %+v", findings)
	}
	if hasCheck(findings, "epic.empty", filled.ID) {
		t.Errorf("populated epic wrongly flagged: %+v", findings)
	}
}

func TestProjectMissingFlagsVanishedRepos(t *testing.T) {
	s := testStore(t)

	gone, _ := s.CreateProject("gone", filepath.Join(t.TempDir(), "nope"), "")

	findings, _ := Run(s, []string{"projects"}, "")
	if !hasCheck(findings, "project.missing", gone.ID) {
		t.Errorf("missing repository not flagged: %+v", findings)
	}
}

func TestRunWithNoScopesRunsEverything(t *testing.T) {
	s := testStore(t)

	s.CreateWikiPage("", "Orphan", "concept", "", "b", "")
	gone, _ := s.CreateProject("gone", filepath.Join(t.TempDir(), "nope"), "")

	findings, err := Run(s, nil, "")
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if !hasCheck(findings, "wiki.orphans", "") {
		t.Errorf("wiki checks did not run: %+v", findings)
	}
	if !hasCheck(findings, "project.missing", gone.ID) {
		t.Errorf("project checks did not run: %+v", findings)
	}
}

func TestRunRejectsUnknownScope(t *testing.T) {
	s := testStore(t)

	if _, err := Run(s, []string{"nonsense"}, ""); err == nil {
		t.Error("unknown scope accepted, want an error")
	}
}

func TestCleanDatabaseHasNoFindings(t *testing.T) {
	s := testStore(t)

	findings, err := Run(s, nil, "")
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if len(findings) != 0 {
		t.Errorf("empty database produced findings: %+v", findings)
	}
}

func TestProjectMissingReportsCheckFailedDistinctly(t *testing.T) {
	s := testStore(t)

	// A path that exists and is a git repo, but git cannot be invoked
	// against it, must not be silently treated as healthy: silence here
	// would give a false clean bill of health for something doctor could
	// not actually verify.
	dir := t.TempDir()
	if err := os.Mkdir(filepath.Join(dir, ".git"), 0o755); err != nil {
		t.Fatalf("Mkdir: %v", err)
	}

	oldPath := os.Getenv("PATH")
	os.Setenv("PATH", "")
	t.Cleanup(func() { os.Setenv("PATH", oldPath) })

	p, _ := s.CreateProject("broken", dir, "")

	findings, err := Run(s, []string{"projects"}, "")
	if err != nil {
		t.Fatalf("Run: %v", err)
	}

	if hasCheck(findings, "project.missing", p.ID) {
		t.Errorf("check-failed project wrongly reported as project.missing: %+v", findings)
	}
	if !hasCheck(findings, "project.unverifiable", p.ID) {
		t.Errorf("check-failed project not reported as unverifiable: %+v", findings)
	}
}

func TestProjectMissingDoesNotFlagAPresentRepo(t *testing.T) {
	s := testStore(t)

	// A project whose path exists (even if it is not a git repository)
	// must not be conflated with one whose path is gone entirely.
	dir := t.TempDir()
	p, _ := s.CreateProject("present", dir, "")

	findings, err := Run(s, []string{"projects"}, "")
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if hasCheck(findings, "project.missing", p.ID) {
		t.Errorf("present repository wrongly flagged as missing: %+v", findings)
	}
}

func TestWikiMissingDoesNotDoubleCountAcrossPages(t *testing.T) {
	s := testStore(t)

	s.CreateWikiPage("", "Hub One", "concept", "", "See [[does-not-exist]].", "")
	s.CreateWikiPage("", "Hub Two", "concept", "", "Also see [[does-not-exist]].", "")

	findings, err := Run(s, []string{"wiki"}, "")
	if err != nil {
		t.Fatalf("Run: %v", err)
	}

	count := 0
	for _, f := range findings {
		if f.Check == "wiki.missing" && f.ID == "does-not-exist" {
			count++
		}
	}
	if count != 1 {
		t.Errorf("wiki.missing reported %d times for the same target across two pages, want 1: %+v",
			count, findings)
	}
}

func TestStoryStrandedRespectsOverriddenThreshold(t *testing.T) {
	s := testStore(t)

	p, _ := s.CreateProject("my-app", "/tmp/my-app", "")
	e, _ := s.CreateEpic(p.ID, "Auth", "")

	original := store.Now
	store.Now = func() time.Time { return time.Now().UTC().AddDate(0, 0, -3) }
	st, _ := s.CreateStory(e.ID, "stuck for a few days", "")
	inProgress := "in-progress"
	s.UpdateStory(st.ID, store.StoryFields{Status: &inProgress})
	store.Now = original

	// Three days old does not clear the default 14-day threshold.
	findings, _ := Run(s, []string{"stories"}, "")
	if hasCheck(findings, "story.stranded", st.ID) {
		t.Fatalf("story flagged stranded under the default threshold: %+v", findings)
	}

	// Lowering StrandedAfterDays must change the outcome for the exact
	// same story, proving the check actually reads the package variable
	// rather than a hardcoded constant.
	originalThreshold := StrandedAfterDays
	StrandedAfterDays = 1
	t.Cleanup(func() { StrandedAfterDays = originalThreshold })

	findings, _ = Run(s, []string{"stories"}, "")
	if !hasCheck(findings, "story.stranded", st.ID) {
		t.Errorf("story not flagged after lowering StrandedAfterDays: %+v", findings)
	}
}

func TestWikiOrphansSelfLinkStillFlagsOrphan(t *testing.T) {
	s := testStore(t)

	// A page's only inbound edge is a link from itself to itself. That is
	// not an inbound link from anything else, so the page is still an
	// orphan in every sense the check cares about.
	page, _ := s.CreateWikiPage("", "Self Referential", "concept", "", "b", "")
	s.AddLink("wiki", page.ID, "wiki", page.ID, "references")

	findings, err := Run(s, []string{"wiki"}, "")
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if !hasCheck(findings, "wiki.orphans", page.ID) {
		t.Errorf("page whose only inbound edge is a self-link not flagged orphan: %+v", findings)
	}
}

func TestWikiUncitedFlagsPageDerivedFromAnotherWikiPage(t *testing.T) {
	s := testStore(t)

	// A derived-from edge whose target is another wiki page, not a
	// source, cites no source at all. wiki.uncited must not treat that
	// edge as satisfying "this page cites a source."
	other, _ := s.CreateWikiPage("", "Other", "concept", "", "b", "")
	page, _ := s.CreateWikiPage("", "Derived From Wiki", "summary", "", "b", "")
	s.AddLink("wiki", page.ID, "wiki", other.ID, "derived-from")

	findings, _ := Run(s, []string{"wiki"}, "")
	if !hasCheck(findings, "wiki.uncited", page.ID) {
		t.Errorf(
			"page derived-from another wiki page (not a source) not flagged uncited: %+v",
			findings)
	}
}

func TestWikiStaleDoesNotFlagCurrentPageWithNoSupersession(t *testing.T) {
	s := testStore(t)

	current, _ := s.CreateWikiPage("", "Current", "concept", "", "b", "")

	findings, _ := Run(s, []string{"wiki"}, "")
	if hasCheck(findings, "wiki.stale", current.ID) {
		t.Errorf("current page with no supersession edge wrongly flagged stale: %+v", findings)
	}
}

func TestWikiStaleDoesNotReflagAlreadySupersededPage(t *testing.T) {
	s := testStore(t)

	old, _ := s.CreateWikiPage("", "Old", "concept", "", "b", "")
	replacement, _ := s.CreateWikiPage("", "New", "concept", "", "b", "")
	s.AddLink("wiki", replacement.ID, "wiki", old.ID, "supersedes")

	superseded := "superseded"
	if _, err := s.UpdateWikiPage(old.ID, store.WikiFields{Status: &superseded}); err != nil {
		t.Fatalf("UpdateWikiPage: %v", err)
	}

	findings, _ := Run(s, []string{"wiki"}, "")
	if hasCheck(findings, "wiki.stale", old.ID) {
		t.Errorf("already-superseded page wrongly re-flagged as stale: %+v", findings)
	}
}

func TestWikiUnprocessedResurfacesAfterTheDerivedPageIsDeleted(t *testing.T) {
	s := testStore(t)

	src, _ := s.CreateSource("", "An article", "article", "text", "")
	page, _ := s.CreateWikiPage("", "Derived", "summary", "", "b", "")
	s.AddLink("wiki", page.ID, "source", src.ID, "derived-from")

	if err := s.DeleteWikiPage(page.ID); err != nil {
		t.Fatalf("DeleteWikiPage: %v", err)
	}

	// The page that vouched for this source is gone, so the source is
	// unprocessed again. A left-behind edge would keep vouching forever.
	findings, _ := Run(s, []string{"wiki"}, "")
	if !hasCheck(findings, "wiki.unprocessed", src.ID) {
		t.Errorf("source not re-reported after its only derived page was deleted: %+v", findings)
	}
}

func TestWikiOrphansResurfaceAfterTheLinkingPageIsDeleted(t *testing.T) {
	s := testStore(t)

	hub, _ := s.CreateWikiPage("", "Hub", "concept", "", "b", "")
	target, _ := s.CreateWikiPage("", "Target", "concept", "", "b", "")
	s.AddLink("wiki", hub.ID, "wiki", target.ID, "references")

	if err := s.DeleteWikiPage(hub.ID); err != nil {
		t.Fatalf("DeleteWikiPage: %v", err)
	}

	findings, _ := Run(s, []string{"wiki"}, "")
	if !hasCheck(findings, "wiki.orphans", target.ID) {
		t.Errorf("page not re-reported orphan after its only inbound link was deleted: %+v", findings)
	}
}

func TestWikiDanglingFlagsEdgesLeakedByAnOlderBinary(t *testing.T) {
	s, path := testStoreAt(t)

	src, _ := s.CreateSource("", "An article", "article", "text", "")
	page, _ := s.CreateWikiPage("", "Derived", "summary", "", "b", "")
	s.AddLink("wiki", page.ID, "source", src.ID, "derived-from")

	findings, _ := Run(s, []string{"wiki"}, "")
	if hasCheck(findings, "wiki.dangling", "") {
		t.Fatalf("healthy graph reported a dangling edge: %+v", findings)
	}

	// Databases written before Delete* cleaned up carry edges like this.
	forceDelete(t, path, "wiki_pages", page.ID)

	findings, _ = Run(s, []string{"wiki"}, "")
	if !hasCheck(findings, "wiki.dangling", "") {
		t.Errorf("leaked edge not reported as wiki.dangling: %+v", findings)
	}
}

// --- wiki.missing must emit a slug a skill can actually create ---

func TestWikiMissingReportsOnlyTheTargetOfALabelledWikilink(t *testing.T) {
	s := testStore(t)
	s.CreateWikiPage("", "Hub", "concept", "", "see [[Target Page|the label]]", "")

	findings, _ := Run(s, []string{"wiki"}, "")

	if !hasCheck(findings, "wiki.missing", "target-page") {
		t.Errorf("wiki.missing did not report the link target: %+v", findings)
	}
	// The label is not part of the target. A finding for the whole
	// "target|label" run names a slug that can never exist, so a skill
	// told to create it creates the wrong page and the real target stays
	// missing forever.
	if hasCheck(findings, "wiki.missing", "target-page-the-label") {
		t.Errorf("wiki.missing reported the label as part of the target: %+v", findings)
	}
}

func TestWikiMissingSkipsWikilinksWithNoTarget(t *testing.T) {
	s := testStore(t)
	s.CreateWikiPage("", "Hub", "concept", "", "an empty one: [[ ]]", "")

	findings, _ := Run(s, []string{"wiki"}, "")

	for _, f := range findings {
		if f.Check == "wiki.missing" {
			t.Errorf("wiki.missing reported %q for [[ ]]; there is no page to create", f.ID)
		}
	}
}

func TestWikiMissingIgnoresWikilinksWithBracketsInTheTarget(t *testing.T) {
	s := testStore(t)
	s.CreateWikiPage("", "Hub", "concept", "", "malformed: [[a[b]c]]", "")

	findings, _ := Run(s, []string{"wiki"}, "")

	// A target containing brackets is not a slug anything could resolve,
	// so the only honest outcome is silence — never a garbled id.
	for _, f := range findings {
		if f.Check == "wiki.missing" {
			t.Errorf("wiki.missing reported %q for [[a[b]c]]: %+v", f.ID, findings)
		}
	}
}

func TestWikiMissingStillReportsAPlainWikilink(t *testing.T) {
	s := testStore(t)
	s.CreateWikiPage("", "Hub", "concept", "", "see [[Auth Model]]", "")

	findings, _ := Run(s, []string{"wiki"}, "")
	if !hasCheck(findings, "wiki.missing", "auth-model") {
		t.Errorf("plain wikilink no longer reported: %+v", findings)
	}
}

// --- a leaked edge must not silence the checks it touches ---

func TestDanglingEdgeDoesNotSuppressOrphansAndUncited(t *testing.T) {
	s, path := testStoreAt(t)

	src, _ := s.CreateSource("", "An article", "article", "text", "")
	hub, _ := s.CreateWikiPage("", "Hub", "concept", "", "b", "")
	target, _ := s.CreateWikiPage("", "Target", "concept", "", "b", "")
	s.AddLink("wiki", hub.ID, "wiki", target.ID, "references")
	s.AddLink("wiki", target.ID, "source", src.ID, "derived-from")

	// A database written by an older binary: hub is gone, its edge is not.
	forceDelete(t, path, "wiki_pages", hub.ID)
	forceDelete(t, path, "sources", src.ID)

	findings, _ := Run(s, []string{"wiki"}, "")

	if !hasCheck(findings, "wiki.orphans", target.ID) {
		t.Errorf("a leaked edge is still vouching for inbound links: %+v", findings)
	}
	if !hasCheck(findings, "wiki.uncited", target.ID) {
		t.Errorf("a leaked edge is still vouching for a citation: %+v", findings)
	}
}

// --- story.stranded reads the overridable clock ---

// TestStoryStrandedBoundary pins the exact edge of the 14-day window.
// It can only be written because the check measures its cutoff from
// store.Now, the same clock the store stamps UpdatedAt from; against
// time.Now the two clocks disagree and no boundary is expressible.
func TestStoryStrandedBoundary(t *testing.T) {
	base := time.Date(2026, 8, 1, 12, 0, 0, 0, time.UTC)

	for _, tc := range []struct {
		name        string
		ageInDays   int
		wantFlagged bool
	}{
		{"one day past the window", 15, true},
		{"exactly at the window", 14, false},
		{"one day inside the window", 13, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			s := testStore(t)
			original := store.Now
			t.Cleanup(func() { store.Now = original })

			p, _ := s.CreateProject("my-app", "/tmp/my-app", "")
			e, _ := s.CreateEpic(p.ID, "Auth", "")

			store.Now = func() time.Time { return base.AddDate(0, 0, -tc.ageInDays) }
			st, _ := s.CreateStory(e.ID, "stuck", "")
			inProgress := "in-progress"
			s.UpdateStory(st.ID, store.StoryFields{Status: &inProgress})

			store.Now = func() time.Time { return base }
			findings, _ := Run(s, []string{"stories"}, "")

			if got := hasCheck(findings, "story.stranded", st.ID); got != tc.wantFlagged {
				t.Errorf("stranded at %d days = %v, want %v: %+v",
					tc.ageInDays, got, tc.wantFlagged, findings)
			}
		})
	}
}

func TestStoryStrandedMessageDatesTheLastTouch(t *testing.T) {
	s := testStore(t)
	original := store.Now
	t.Cleanup(func() { store.Now = original })

	p, _ := s.CreateProject("my-app", "/tmp/my-app", "")
	e, _ := s.CreateEpic(p.ID, "Auth", "")

	store.Now = func() time.Time { return time.Now().UTC().AddDate(0, 0, -30) }
	st, _ := s.CreateStory(e.ID, "stuck", "")
	inProgress := "in-progress"
	s.UpdateStory(st.ID, store.StoryFields{Status: &inProgress})
	store.Now = original

	findings, _ := Run(s, []string{"stories"}, "")
	for _, f := range findings {
		if f.Check != "story.stranded" || f.ID != st.ID {
			continue
		}
		// UpdatedAt moves on any edit, so it dates the last touch, not
		// the move into in-progress. The message must not claim more.
		if !strings.Contains(f.Message, "untouched since") {
			t.Errorf("stranded message = %q, want it to say 'untouched since'", f.Message)
		}
		return
	}
	t.Fatalf("no story.stranded finding for %s: %+v", st.ID, findings)
}

// --- -p scopes the report ---

func TestRunScopesWikiPagesToTheNamedProject(t *testing.T) {
	s := testStore(t)

	mine, _ := s.CreateProject("mine", "/tmp/mine", "")
	theirs, _ := s.CreateProject("theirs", "/tmp/theirs", "")
	ours, _ := s.CreateWikiPage("", "Ours", "concept", "", "b", mine.ID)
	notOurs, _ := s.CreateWikiPage("", "Not Ours", "concept", "", "b", theirs.ID)

	findings, err := Run(s, []string{"wiki"}, mine.ID)
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if !hasCheck(findings, "wiki.orphans", ours.ID) {
		t.Errorf("the named project's own page was not reported: %+v", findings)
	}
	if hasCheck(findings, "wiki.orphans", notOurs.ID) {
		t.Errorf("another project's page leaked into a scoped report: %+v", findings)
	}
}

func TestRunScopesStoriesAndEpicsToTheNamedProject(t *testing.T) {
	s := testStore(t)

	mine, _ := s.CreateProject("mine", "/tmp/mine", "")
	theirs, _ := s.CreateProject("theirs", "/tmp/theirs", "")
	myEpic, _ := s.CreateEpic(mine.ID, "Mine", "")
	theirEpic, _ := s.CreateEpic(theirs.ID, "Theirs", "")

	myStory, _ := s.CreateStory(myEpic.ID, "mine", "")
	theirStory, _ := s.CreateStory(theirEpic.ID, "theirs", "")
	done := "done"
	s.UpdateStory(myStory.ID, store.StoryFields{Status: &done})
	s.UpdateStory(theirStory.ID, store.StoryFields{Status: &done})

	findings, _ := Run(s, []string{"stories"}, mine.ID)

	if !hasCheck(findings, "story.planless", myStory.ID) {
		t.Errorf("the named project's story was not reported: %+v", findings)
	}
	if hasCheck(findings, "story.planless", theirStory.ID) {
		t.Errorf("another project's story leaked into a scoped report: %+v", findings)
	}
	if hasCheck(findings, "epic.empty", theirEpic.ID) {
		t.Errorf("another project's epic leaked into a scoped report: %+v", findings)
	}
}

func TestRunScopesProjectChecksToTheNamedProject(t *testing.T) {
	s := testStore(t)

	// Two projects whose paths do not exist: both are project.missing.
	mine, _ := s.CreateProject("mine", filepath.Join(t.TempDir(), "gone-mine"), "")
	theirs, _ := s.CreateProject("theirs", filepath.Join(t.TempDir(), "gone-theirs"), "")

	findings, _ := Run(s, []string{"projects"}, mine.ID)

	if !hasCheck(findings, "project.missing", mine.ID) {
		t.Errorf("the named project was not reported: %+v", findings)
	}
	if hasCheck(findings, "project.missing", theirs.ID) {
		t.Errorf("another project leaked into a scoped report: %+v", findings)
	}
}

func TestRunRejectsAnUnknownProject(t *testing.T) {
	s := testStore(t)

	_, err := Run(s, []string{"stories"}, "no-such-project")
	if !errors.Is(err, store.ErrNotFound) {
		t.Errorf("Run with an unknown project: err = %v, want ErrNotFound", err)
	}
}

func TestRunRejectsAnUnknownScopeAsBadInput(t *testing.T) {
	s := testStore(t)

	_, err := Run(s, []string{"nonsense"}, "")
	if !errors.Is(err, store.ErrInvalid) {
		t.Errorf("Run with an unknown scope: err = %v, want ErrInvalid", err)
	}
}
