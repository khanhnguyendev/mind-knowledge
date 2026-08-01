package cli

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/khanhnguyendev/mind-knowledge/internal/render"
)

func init() {
	var kinds []string

	searchCmd := &cobra.Command{
		Use:   "search <query>",
		Short: "Full-text search across stories, wiki pages, and sources",
		Long: "Search is machine-wide and does not take -p/--project. " +
			"Sources belong to no project, so a project-scoped search " +
			"would have to silently drop a third of the index; use " +
			"--kind to narrow instead.",
		Args: cobra.MinimumNArgs(1),
		// Rejecting -p rather than ignoring it: see rejectProjectFlag.
		PersistentPreRunE: crossProjectPreRun("search results"),
		RunE: func(cmd *cobra.Command, args []string) error {
			s, err := OpenStore()
			if err != nil {
				return err
			}
			defer s.Close()

			hits, err := s.Search(strings.Join(args, " "), kinds, LimitFlag())
			if err != nil {
				return err
			}
			if JSONMode() {
				return render.JSON(os.Stdout, hits)
			}
			for _, h := range hits {
				fmt.Printf("%-6s  %s  %s\n", h.Kind, h.ID, h.Title)
				if h.Snippet != "" {
					fmt.Printf("        %s\n", strings.ReplaceAll(h.Snippet, "\n", " "))
				}
			}
			return nil
		},
	}

	searchCmd.Flags().StringSliceVar(&kinds, "kind", nil,
		"restrict to story, wiki, or source; repeatable")

	Root.AddCommand(searchCmd)
}
