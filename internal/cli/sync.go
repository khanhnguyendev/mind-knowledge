package cli

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/khanhnguyendev/mind-knowledge/internal/model"
	"github.com/khanhnguyendev/mind-knowledge/internal/render"
	mksync "github.com/khanhnguyendev/mind-knowledge/internal/sync"
)

func init() {
	syncCmd := &cobra.Command{
		Use:   "sync",
		Short: "Reconcile registered projects against the filesystem",
		Long: "Reports whether each project's path still exists, is still a " +
			"git repository, and still points at the recorded remote. It " +
			"changes nothing.",
		RunE: func(cmd *cobra.Command, args []string) error {
			s, err := OpenStore()
			if err != nil {
				return err
			}
			defer s.Close()

			var projects []model.Project
			if name := ProjectFlag(); name != "" {
				p, err := s.GetProject(name)
				if err != nil {
					return err
				}
				projects = []model.Project{*p}
			} else {
				projects, err = s.ListProjects("", 0)
				if err != nil {
					return err
				}
			}

			results := mksync.Run(s, projects)
			if JSONMode() {
				return render.JSON(os.Stdout, results)
			}

			rows := make([][]string, 0, len(results))
			for _, r := range results {
				detail := r.Detail
				if detail == "" && r.Branch != "" {
					detail = fmt.Sprintf("%s @ %s", r.Branch, shortHead(r.Head))
				}
				rows = append(rows, []string{r.Project.Name, r.State, detail})
			}
			render.Table(os.Stdout, []string{"PROJECT", "STATE", "DETAIL"}, rows)
			return nil
		},
	}

	Root.AddCommand(syncCmd)
}

// shortHead abbreviates a commit hash for display.
func shortHead(head string) string {
	if len(head) > 7 {
		return head[:7]
	}
	return head
}
