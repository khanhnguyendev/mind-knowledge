package cli

import (
	"os"

	"github.com/spf13/cobra"

	"github.com/khanhnguyendev/mind-knowledge/internal/model"
	"github.com/khanhnguyendev/mind-knowledge/internal/render"
	"github.com/khanhnguyendev/mind-knowledge/internal/store"
)

func init() {
	var showAll bool
	var filterStatus string

	boardCmd := &cobra.Command{
		Use:   "board",
		Short: "Show stories grouped by epic and project",
		// board takes no positional arguments. Leaving Args nil would let
		// cobra's legacyArgs silently accept and ignore any stray one
		// (board has no subcommands, so legacyArgs never even inspects
		// them) — the same silent-success shape as the group-command bug
		// this change accompanies, reached via a different cobra path.
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			s, err := OpenStore()
			if err != nil {
				return err
			}
			defer s.Close()

			projects, err := boardProjects(s, showAll)
			if err != nil {
				return err
			}

			out := make([]render.BoardProject, 0, len(projects))
			for _, p := range projects {
				epics, err := s.ListEpics(p.ID, "", 0)
				if err != nil {
					return err
				}

				bp := render.BoardProject{Name: p.Name, Epics: []render.BoardEpic{}}
				for _, e := range epics {
					stories, err := s.ListStories(store.StoryFilter{
						EpicID: e.ID,
						Status: filterStatus,
					})
					if err != nil {
						return err
					}

					be := render.BoardEpic{
						ID:      e.ID,
						Title:   e.Title,
						Stories: []render.BoardStory{},
					}
					for _, st := range stories {
						be.Stories = append(be.Stories, render.BoardStory{
							ID:     st.ID,
							Status: st.Status,
							Title:  st.Title,
						})
					}
					bp.Epics = append(bp.Epics, be)
				}
				out = append(out, bp)
			}

			if JSONMode() {
				return render.JSON(os.Stdout, out)
			}
			render.Board(os.Stdout, out)
			return nil
		},
	}

	boardCmd.Flags().BoolVar(&showAll, "all", false,
		"include paused and archived projects")
	boardCmd.Flags().StringVar(&filterStatus, "status", "",
		"show only stories in this status")

	Root.AddCommand(boardCmd)
}

// boardProjects resolves which projects the board covers: the one named by
// -p, or every active project, or every project when --all is given.
func boardProjects(s *store.Store, showAll bool) ([]model.Project, error) {
	if name := ProjectFlag(); name != "" {
		p, err := s.GetProject(name)
		if err != nil {
			return nil, err
		}
		return []model.Project{*p}, nil
	}
	if showAll {
		return s.ListProjects("", 0)
	}
	return s.ListProjects("active", 0)
}
