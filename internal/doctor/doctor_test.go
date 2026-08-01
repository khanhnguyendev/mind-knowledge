package doctor

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/khanhnguyendev/mind-knowledge/internal/store"
)

func testStore(t *testing.T) *store.Store {
	t.Helper()
	s, err := store.Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	t.Cleanup(func() { s.Close() })
	return s
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

	findings, err := Run(s, []string{"wiki"})
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

	findings, _ := Run(s, []string{"wiki"})
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

	findings, _ := Run(s, []string{"wiki"})
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

	findings, _ := Run(s, []string{"wiki"})
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

	findings, _ := Run(s, []string{"wiki"})

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

	findings, _ := Run(s, []string{"stories"})
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

	findings, _ := Run(s, []string{"stories"})
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

	findings, _ := Run(s, []string{"stories"})
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

	findings, _ := Run(s, []string{"projects"})
	if !hasCheck(findings, "project.missing", gone.ID) {
		t.Errorf("missing repository not flagged: %+v", findings)
	}
}

func TestRunWithNoScopesRunsEverything(t *testing.T) {
	s := testStore(t)

	s.CreateWikiPage("", "Orphan", "concept", "", "b", "")
	gone, _ := s.CreateProject("gone", filepath.Join(t.TempDir(), "nope"), "")

	findings, err := Run(s, nil)
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

	if _, err := Run(s, []string{"nonsense"}); err == nil {
		t.Error("unknown scope accepted, want an error")
	}
}

func TestCleanDatabaseHasNoFindings(t *testing.T) {
	s := testStore(t)

	findings, err := Run(s, nil)
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

	findings, err := Run(s, []string{"projects"})
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

	findings, err := Run(s, []string{"projects"})
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

	findings, err := Run(s, []string{"wiki"})
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
	findings, _ := Run(s, []string{"stories"})
	if hasCheck(findings, "story.stranded", st.ID) {
		t.Fatalf("story flagged stranded under the default threshold: %+v", findings)
	}

	// Lowering StrandedAfterDays must change the outcome for the exact
	// same story, proving the check actually reads the package variable
	// rather than a hardcoded constant.
	originalThreshold := StrandedAfterDays
	StrandedAfterDays = 1
	t.Cleanup(func() { StrandedAfterDays = originalThreshold })

	findings, _ = Run(s, []string{"stories"})
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

	findings, err := Run(s, []string{"wiki"})
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

	findings, _ := Run(s, []string{"wiki"})
	if !hasCheck(findings, "wiki.uncited", page.ID) {
		t.Errorf(
			"page derived-from another wiki page (not a source) not flagged uncited: %+v",
			findings)
	}
}

func TestWikiStaleDoesNotFlagCurrentPageWithNoSupersession(t *testing.T) {
	s := testStore(t)

	current, _ := s.CreateWikiPage("", "Current", "concept", "", "b", "")

	findings, _ := Run(s, []string{"wiki"})
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

	findings, _ := Run(s, []string{"wiki"})
	if hasCheck(findings, "wiki.stale", old.ID) {
		t.Errorf("already-superseded page wrongly re-flagged as stale: %+v", findings)
	}
}
