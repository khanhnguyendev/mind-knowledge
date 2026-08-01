package render

import (
	"fmt"
	"io"
)

// BoardStory is one story as the board shows it.
type BoardStory struct {
	ID     string `json:"id"`
	Status string `json:"status"`
	Title  string `json:"title"`
}

// BoardEpic groups stories under their epic.
type BoardEpic struct {
	ID      string       `json:"id"`
	Title   string       `json:"title"`
	Stories []BoardStory `json:"stories"`
}

// BoardProject groups epics under their project.
type BoardProject struct {
	Name  string      `json:"name"`
	Epics []BoardEpic `json:"epics"`
}

// Board writes the board as indented plain text. JSON callers marshal the
// same []BoardProject instead, so both modes describe one shape.
func Board(w io.Writer, projects []BoardProject) {
	if len(projects) == 0 {
		fmt.Fprintln(w, "no projects registered")
		return
	}

	for i, p := range projects {
		if i > 0 {
			fmt.Fprintln(w)
		}
		fmt.Fprintln(w, p.Name)

		if len(p.Epics) == 0 {
			fmt.Fprintln(w, "  (no epics)")
			continue
		}

		for _, e := range p.Epics {
			fmt.Fprintf(w, "  %s (epic %s)\n", e.Title, e.ID)
			if len(e.Stories) == 0 {
				fmt.Fprintln(w, "    (no stories)")
				continue
			}

			statusWidth := 0
			for _, s := range e.Stories {
				if len(s.Status) > statusWidth {
					statusWidth = len(s.Status)
				}
			}
			for _, s := range e.Stories {
				fmt.Fprintf(w, "    %-*s  %s  %s\n", statusWidth, s.Status, s.ID, s.Title)
			}
		}
	}
}
