// Package doctor reports drift. It never repairs anything: the mk binary
// does not enforce workflow at write time, so skipped steps surface here
// instead.
package doctor

import (
	"fmt"
	"regexp"
	"time"

	"github.com/khanhnguyendev/mind-knowledge/internal/model"
	"github.com/khanhnguyendev/mind-knowledge/internal/store"
	mksync "github.com/khanhnguyendev/mind-knowledge/internal/sync"
)

// StrandedAfterDays is how long a story may sit in-progress before
// story.stranded reports it.
var StrandedAfterDays = 14

// Finding is one reported problem.
type Finding struct {
	Check   string `json:"check"`
	Kind    string `json:"kind"`
	ID      string `json:"id"`
	Message string `json:"message"`
}

// wikilinkPattern matches [[target]] and [[target|label]] references
// inside page bodies, capturing only the target.
//
// The target must not itself contain a bracket or a pipe. That is what
// keeps the capture actionable: a finding names a slug a skill can go and
// create, so capturing "Target Page|the label" (which slugifies to
// target-page-the-label, a page that can never legitimately exist) would
// send the skill off to create the wrong page while the real target stayed
// missing and the finding re-fired forever.
var wikilinkPattern = regexp.MustCompile(`\[\[([^\[\]|]+)(?:\|[^\[\]]*)?\]\]`)

// Run executes the checks named by scopes. An empty scopes slice runs
// everything. A non-empty projectID restricts the checks to the entities
// that belong to that project; see each check for what that means, since
// sources and links belong to no project at all.
func Run(s *store.Store, scopes []string, projectID string) ([]Finding, error) {
	if len(scopes) == 0 {
		scopes = []string{"wiki", "stories", "projects"}
	}

	findings := []Finding{}
	for _, scope := range scopes {
		var (
			got []Finding
			err error
		)
		switch scope {
		case "wiki":
			got, err = checkWiki(s, projectID)
		case "stories":
			got, err = checkStories(s, projectID)
		case "projects":
			got, err = checkProjects(s, projectID)
		default:
			return nil, fmt.Errorf(
				"%w: unknown doctor scope %q (want wiki, stories, or projects)",
				store.ErrInvalid, scope)
		}
		if err != nil {
			return nil, err
		}
		findings = append(findings, got...)
	}
	return findings, nil
}

func checkWiki(s *store.Store, projectID string) ([]Finding, error) {
	// The pages reported on are the ones in scope.
	pages, err := s.ListWikiPages("", "", projectID, 0)
	if err != nil {
		return nil, err
	}

	// Everything a page is judged *against*, though, stays machine-wide.
	// A slug resolves across the whole wiki, so a [[wikilink]] into
	// another project's page is not missing; and an inbound link from
	// another project's page is still an inbound link.
	allPages := pages
	if projectID != "" {
		allPages, err = s.ListWikiPages("", "", "", 0)
		if err != nil {
			return nil, err
		}
	}

	// Sources carry no project at all, so wiki.unprocessed is reported
	// machine-wide even under -p: an uningested source is a debt against
	// the wiki as a whole, and scoping it away would leave no way to see
	// it.
	sources, err := s.ListSources("", 0)
	if err != nil {
		return nil, err
	}
	links, err := s.ListLinks("", "", "", "", "")
	if err != nil {
		return nil, err
	}
	dangling, err := s.DanglingLinks()
	if err != nil {
		return nil, err
	}
	isDangling := make(map[model.Link]bool, len(dangling))
	for _, l := range dangling {
		isDangling[l] = true
	}

	// Build the link maps once, from links, so wiki.uncited and
	// wiki.unprocessed — which reason about the same derived-from edges
	// from opposite ends (the wiki page side and the source side) — read
	// off exactly one condition (derived-from, wiki -> source) instead of
	// two independently written ones that could drift apart.
	inbound := map[string]bool{}
	derivedFrom := map[string]bool{}
	supersededPages := map[string]bool{}
	citedSources := map[string]bool{}

	for _, l := range links {
		// An edge whose endpoint no longer exists must not vouch for the
		// end that survived. Left in, one leaked edge silently turns off
		// wiki.orphans, wiki.uncited, and wiki.unprocessed for its
		// surviving endpoint — for good.
		if isDangling[l] {
			continue
		}
		// A link from a page to itself is not an inbound link from
		// anything else, so it must not count toward "has inbound
		// links."
		if l.ToKind == "wiki" && !(l.FromKind == "wiki" && l.FromID == l.ToID) {
			inbound[l.ToID] = true
		}
		// Only a derived-from edge that actually terminates at a source
		// counts as citing a source. A derived-from edge to another wiki
		// page cites no source at all, so derivedFrom and citedSources
		// must be set together, from the same condition, or the two
		// checks below can disagree about what "cited" means.
		if l.Relation == "derived-from" && l.FromKind == "wiki" && l.ToKind == "source" {
			derivedFrom[l.FromID] = true
			citedSources[l.ToID] = true
		}
		if l.Relation == "supersedes" && l.ToKind == "wiki" {
			supersededPages[l.ToID] = true
		}
	}

	knownSlugs := map[string]bool{}
	for _, p := range allPages {
		knownSlugs[p.Slug] = true
	}

	findings := []Finding{}
	reportedMissing := map[string]bool{}

	// Dangling edges are integrity damage, not project drift: with an
	// endpoint gone there is nothing left to tell us whose project it
	// was, so they are reported whatever -p says. mk's own Delete* paths
	// no longer produce them; these are what a database written by an
	// older binary still carries.
	for _, l := range dangling {
		findings = append(findings, Finding{
			Check: "wiki.dangling", Kind: "link",
			ID: l.FromKind + ":" + l.FromID,
			Message: fmt.Sprintf(
				"edge %s:%s -[%s]-> %s:%s names an entity that no longer exists",
				l.FromKind, l.FromID, l.Relation, l.ToKind, l.ToID),
		})
	}

	for _, p := range pages {
		if !inbound[p.ID] {
			findings = append(findings, Finding{
				Check: "wiki.orphans", Kind: "wiki", ID: p.ID,
				Message: fmt.Sprintf("%s has no inbound links", p.Slug),
			})
		}
		if supersededPages[p.ID] && p.Status == "current" {
			findings = append(findings, Finding{
				Check: "wiki.stale", Kind: "wiki", ID: p.ID,
				Message: fmt.Sprintf(
					"%s is superseded by another page but still marked current", p.Slug),
			})
		}
		if !derivedFrom[p.ID] {
			findings = append(findings, Finding{
				Check: "wiki.uncited", Kind: "wiki", ID: p.ID,
				Message: fmt.Sprintf("%s cites no source", p.Slug),
			})
		}

		for _, match := range wikilinkPattern.FindAllStringSubmatch(p.Body, -1) {
			target := model.Slugify(match[1])
			// [[ ]] and friends slugify to nothing. There is no page a
			// skill could create to satisfy such a finding, so reporting
			// one only produces noise that can never be cleared.
			if target == "" {
				continue
			}
			if knownSlugs[target] || reportedMissing[target] {
				continue
			}
			reportedMissing[target] = true
			findings = append(findings, Finding{
				Check: "wiki.missing", Kind: "wiki", ID: target,
				Message: fmt.Sprintf("%s links to [[%s]], which has no page", p.Slug, match[1]),
			})
		}
	}

	for _, src := range sources {
		if !citedSources[src.ID] {
			findings = append(findings, Finding{
				Check: "wiki.unprocessed", Kind: "source", ID: src.ID,
				Message: fmt.Sprintf("%q has no page derived from it", src.Title),
			})
		}
	}

	return findings, nil
}

func checkStories(s *store.Store, projectID string) ([]Finding, error) {
	stories, err := s.ListStories(store.StoryFilter{ProjectID: projectID})
	if err != nil {
		return nil, err
	}
	epics, err := s.ListEpics(projectID, "", 0)
	if err != nil {
		return nil, err
	}

	// store.Now, not time.Now: the store stamps UpdatedAt from it, so a
	// test that moves the clock to age a story must move the same clock
	// the cutoff is measured from.
	cutoff := store.Now().UTC().AddDate(0, 0, -StrandedAfterDays)
	populated := map[string]bool{}
	findings := []Finding{}

	for _, st := range stories {
		populated[st.EpicID] = true

		if st.Status == "done" && st.Plan == "" {
			findings = append(findings, Finding{
				Check: "story.planless", Kind: "story", ID: st.ID,
				Message: fmt.Sprintf("%q is done but has no plan", st.Title),
			})
		}
		if st.Status == "in-progress" {
			updated, err := time.Parse(time.RFC3339, st.UpdatedAt)
			if err == nil && updated.Before(cutoff) {
				findings = append(findings, Finding{
					Check: "story.stranded", Kind: "story", ID: st.ID,
					// "untouched since", not "in-progress since":
					// UpdatedAt moves on any edit, so it dates the last
					// change of any kind, not the move into in-progress.
					Message: fmt.Sprintf(
						"%q is in-progress and untouched since %s", st.Title, st.UpdatedAt),
				})
			}
		}
	}

	for _, e := range epics {
		if !populated[e.ID] {
			findings = append(findings, Finding{
				Check: "epic.empty", Kind: "epic", ID: e.ID,
				Message: fmt.Sprintf("%q has no stories", e.Title),
			})
		}
	}

	return findings, nil
}

func checkProjects(s *store.Store, projectID string) ([]Finding, error) {
	var projects []model.Project
	if projectID != "" {
		p, err := s.GetProject(projectID)
		if err != nil {
			return nil, err
		}
		projects = []model.Project{*p}
	} else {
		var err error
		projects, err = s.ListProjects("", 0)
		if err != nil {
			return nil, err
		}
	}

	findings := []Finding{}
	for _, p := range projects {
		switch mksync.Inspect(p).State {
		case mksync.StateMissing:
			findings = append(findings, Finding{
				Check: "project.missing", Kind: "project", ID: p.ID,
				Message: fmt.Sprintf("%s no longer exists at %s", p.Name, p.RepoPath),
			})
		case mksync.StateCheckFailed:
			// git could not be run at all, so we have no evidence either
			// way about the project's health. Reporting this as
			// project.missing would claim a fact we don't have; staying
			// silent would give a false clean bill of health for
			// something doctor could not actually verify. Report it as
			// its own check instead.
			findings = append(findings, Finding{
				Check: "project.unverifiable", Kind: "project", ID: p.ID,
				Message: fmt.Sprintf(
					"%s could not be checked (git did not run) at %s", p.Name, p.RepoPath),
			})
		}
	}
	return findings, nil
}
