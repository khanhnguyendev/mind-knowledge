package cli

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/khanhnguyendev/mind-knowledge/internal/render"
)

func init() {
	logCmd := &cobra.Command{
		Use:   "log",
		Short: "Record and read what happened",
		Long: "The log is how the next session learns what the last one did. " +
			"Every skill appends an entry as its final act.",
	}

	// add
	var addKind, addProject, addRef, addSummary string
	addCmd := &cobra.Command{
		Use:   "add",
		Short: "Append a log entry",
		RunE: func(cmd *cobra.Command, args []string) error {
			s, err := OpenStore()
			if err != nil {
				return err
			}
			defer s.Close()

			project := addProject
			if project == "" {
				project = ProjectFlag()
			}

			entry, err := s.AddLog(addKind, project, addRef, addSummary)
			if err != nil {
				return err
			}
			if JSONMode() {
				return render.JSON(os.Stdout, entry)
			}
			return nil
		},
	}
	addCmd.Flags().StringVar(&addKind, "kind", "",
		"what happened: init, brainstorm, ingest, query, lint, move, done (required)")
	addCmd.Flags().StringVar(&addProject, "project", "", "project this concerns")
	addCmd.Flags().StringVar(&addRef, "ref", "", "id of the entity touched")
	addCmd.Flags().StringVar(&addSummary, "summary", "", "one line describing it (required)")
	addCmd.MarkFlagRequired("kind")
	addCmd.MarkFlagRequired("summary")

	// ls
	var lsKind string
	var lsTail int
	lsCmd := &cobra.Command{
		Use:   "ls",
		Short: "Show log entries, newest first",
		RunE: func(cmd *cobra.Command, args []string) error {
			s, err := OpenStore()
			if err != nil {
				return err
			}
			defer s.Close()

			limit := lsTail
			if limit == 0 {
				limit = LimitFlag()
			}

			entries, err := s.ListLog(lsKind, ProjectFlag(), limit)
			if err != nil {
				return err
			}
			if JSONMode() {
				return render.JSON(os.Stdout, entries)
			}
			for _, e := range entries {
				fmt.Printf("%s  %-10s  %s\n", e.TS, e.Kind, e.Summary)
			}
			return nil
		},
	}
	lsCmd.Flags().StringVar(&lsKind, "kind", "", "filter by kind")
	lsCmd.Flags().IntVar(&lsTail, "tail", 0, "show only the newest N entries")

	logCmd.AddCommand(addCmd, lsCmd)
	Root.AddCommand(logCmd)
}
