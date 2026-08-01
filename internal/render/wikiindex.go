package render

import (
	"fmt"
	"io"
	"sort"

	"github.com/khanhnguyendev/mind-knowledge/internal/model"
)

// WikiIndex writes the wiki catalog as markdown, grouped by kind. This is
// what a skill reads first when answering a question, before drilling into
// individual pages.
func WikiIndex(w io.Writer, pages []model.WikiPage) {
	fmt.Fprintln(w, "# Wiki Index")

	if len(pages) == 0 {
		fmt.Fprintln(w)
		fmt.Fprintln(w, "No pages yet.")
		return
	}

	byKind := map[string][]model.WikiPage{}
	for _, p := range pages {
		byKind[p.Kind] = append(byKind[p.Kind], p)
	}

	kinds := make([]string, 0, len(byKind))
	for kind := range byKind {
		kinds = append(kinds, kind)
	}
	sort.Strings(kinds)

	for _, kind := range kinds {
		group := byKind[kind]
		sort.Slice(group, func(i, j int) bool { return group[i].Slug < group[j].Slug })

		fmt.Fprintf(w, "\n## %s\n\n", kind)
		for _, p := range group {
			line := fmt.Sprintf("- [%s](%s)", p.Title, p.Slug)
			if p.Summary != "" {
				line += " — " + p.Summary
			}
			if p.Status != "current" {
				line += fmt.Sprintf(" _(%s)_", p.Status)
			}
			fmt.Fprintln(w, line)
		}
	}
}
