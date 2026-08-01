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

// wikilinkPattern matches [[slug]] references inside page bodies.
var wikilinkPattern = regexp.MustCompile(`\[\[([^\]]+)\]\]`)

// Run executes the checks named by scopes. An empty scopes slice runs
// everything.
func Run(s *store.Store, scopes []string) ([]Finding, error) {
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
			got, err = checkWiki(s)
		case "stories":
			got, err = checkStories(s)
		case "projects":
			got, err = checkProjects(s)
		default:
			return nil, fmt.Errorf(
				"unknown doctor scope %q (want wiki, stories, or projects)", scope)
		}
		if err != nil {
			return nil, err
		}
		findings = append(findings, got...)
	}
	return findings, nil
}

func checkWiki(s *store.Store) ([]Finding, error) {
	pages, err := s.ListWikiPages("", "", "", 0)
	if err != nil {
		return nil, err
	}
	sources, err := s.ListSources("", 0)
	if err != nil {
		return nil, err
	}
	links, err := s.ListLinks("", "", "", "", "")
	if err != nil {
		return nil, err
	}

	// Build the link maps once, from links, so wiki.uncited and
	// wiki.unprocessed — which reason about the same derived-from edges
	// from opposite ends (the wiki page side and the source side) — can
	// never quietly disagree about which edges exist.
	inbound := map[string]bool{}
	derivedFrom := map[string]bool{}
	supersededPages := map[string]bool{}
	citedSources := map[string]bool{}

	for _, l := range links {
		if l.ToKind == "wiki" {
			inbound[l.ToID] = true
		}
		if l.Relation == "derived-from" && l.FromKind == "wiki" {
			derivedFrom[l.FromID] = true
			if l.ToKind == "source" {
				citedSources[l.ToID] = true
			}
		}
		if l.Relation == "supersedes" && l.ToKind == "wiki" {
			supersededPages[l.ToID] = true
		}
	}

	knownSlugs := map[string]bool{}
	for _, p := range pages {
		knownSlugs[p.Slug] = true
	}

	findings := []Finding{}
	reportedMissing := map[string]bool{}

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

func checkStories(s *store.Store) ([]Finding, error) {
	stories, err := s.ListStories(store.StoryFilter{})
	if err != nil {
		return nil, err
	}
	epics, err := s.ListEpics("", "", 0)
	if err != nil {
		return nil, err
	}

	cutoff := time.Now().UTC().AddDate(0, 0, -StrandedAfterDays)
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
					Message: fmt.Sprintf(
						"%q has been in-progress since %s", st.Title, st.UpdatedAt),
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

func checkProjects(s *store.Store) ([]Finding, error) {
	projects, err := s.ListProjects("", 0)
	if err != nil {
		return nil, err
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
