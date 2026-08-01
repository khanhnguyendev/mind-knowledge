package cli

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/khanhnguyendev/mind-knowledge/internal/render"
	"github.com/khanhnguyendev/mind-knowledge/internal/store"
)

// wikiCmd is package-level so Task 10 can attach the index subcommand from
// a second init in the same file, keeping the index command's flags out of
// the CRUD commands' closure below.
var wikiCmd = &cobra.Command{
	Use:   "wiki",
	Short: "Write and read the wiki",
}

func init() {
	// add
	var addSlug, addTitle, addKind, addSummary, addBody, addProject string
	addCmd := &cobra.Command{
		Use:   "add",
		Short: "Add a wiki page",
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

			p, err := s.CreateWikiPage(addSlug, addTitle, addKind, addSummary, addBody, project)
			if err != nil {
				return err
			}
			return emitCreated(p, p.ID)
		},
	}
	addCmd.Flags().StringVar(&addSlug, "slug", "", "explicit slug (default: derived from title)")
	addCmd.Flags().StringVar(&addTitle, "title", "", "page title (required)")
	addCmd.Flags().StringVar(&addKind, "kind", "concept",
		"summary, concept, entity, decision, spec, synthesis, or comparison")
	addCmd.Flags().StringVar(&addSummary, "summary", "", "one-line summary shown in the index")
	addCmd.Flags().StringVar(&addBody, "body", "", "page body in markdown")
	addCmd.Flags().StringVar(&addProject, "project", "",
		"scope the page to a project (default: cross-project)")
	addCmd.MarkFlagRequired("title")

	// ls
	var lsKind, lsStatus string
	lsCmd := &cobra.Command{
		Use:   "ls",
		Short: "List wiki pages",
		RunE: func(cmd *cobra.Command, args []string) error {
			s, err := OpenStore()
			if err != nil {
				return err
			}
			defer s.Close()

			pages, err := s.ListWikiPages(lsKind, lsStatus, ProjectFlag(), LimitFlag())
			if err != nil {
				return err
			}
			if JSONMode() {
				return render.JSON(os.Stdout, pages)
			}
			rows := make([][]string, 0, len(pages))
			for _, p := range pages {
				rows = append(rows, []string{p.Slug, p.Kind, p.Status, p.Summary})
			}
			render.Table(os.Stdout, []string{"SLUG", "KIND", "STATUS", "SUMMARY"}, rows)
			return nil
		},
	}
	lsCmd.Flags().StringVar(&lsKind, "kind", "", "filter by kind")
	lsCmd.Flags().StringVar(&lsStatus, "status", "", "filter by status")

	// view
	viewCmd := &cobra.Command{
		Use:   "view <id-or-slug>",
		Short: "Show one wiki page",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			s, err := OpenStore()
			if err != nil {
				return err
			}
			defer s.Close()

			p, err := s.GetWikiPage(args[0])
			if err != nil {
				return err
			}
			if JSONMode() {
				return render.JSON(os.Stdout, p)
			}
			// Ordered output: unlike story.go's view, this never ranges
			// over a map, so the field order is stable between runs.
			fmt.Printf("# %s\n", p.Title)
			fmt.Printf("slug: %s  kind: %s  status: %s\n", p.Slug, p.Kind, p.Status)
			if p.Summary != "" {
				fmt.Printf("summary: %s\n", p.Summary)
			}
			if p.Body != "" {
				fmt.Printf("\n%s\n", p.Body)
			}
			return nil
		},
	}

	// edit
	var editSlug, editTitle, editKind, editSummary, editBody, editStatus, editProject string
	editCmd := &cobra.Command{
		Use:   "edit <id-or-slug>",
		Short: "Change a wiki page",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			s, err := OpenStore()
			if err != nil {
				return err
			}
			defer s.Close()

			f := store.WikiFields{}
			if cmd.Flags().Changed("slug") {
				f.Slug = &editSlug
			}
			if cmd.Flags().Changed("title") {
				f.Title = &editTitle
			}
			if cmd.Flags().Changed("kind") {
				f.Kind = &editKind
			}
			if cmd.Flags().Changed("summary") {
				f.Summary = &editSummary
			}
			if cmd.Flags().Changed("body") {
				f.Body = &editBody
			}
			if cmd.Flags().Changed("status") {
				f.Status = &editStatus
			}
			if cmd.Flags().Changed("project") {
				f.ProjectID = &editProject
			}

			p, err := s.UpdateWikiPage(args[0], f)
			if err != nil {
				return err
			}
			return emitCreated(p, p.ID)
		},
	}
	editCmd.Flags().StringVar(&editSlug, "slug", "", "new slug")
	editCmd.Flags().StringVar(&editTitle, "title", "", "new title")
	editCmd.Flags().StringVar(&editKind, "kind", "", "new kind")
	editCmd.Flags().StringVar(&editSummary, "summary", "", "new one-line summary")
	editCmd.Flags().StringVar(&editBody, "body", "", "new body")
	editCmd.Flags().StringVar(&editStatus, "status", "",
		"new status: current, stale, or superseded")
	editCmd.Flags().StringVar(&editProject, "project", "",
		"scope to a project, or empty to make it cross-project")

	// rm
	rmCmd := &cobra.Command{
		Use:   "rm <id-or-slug>",
		Short: "Remove a wiki page",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			s, err := OpenStore()
			if err != nil {
				return err
			}
			defer s.Close()

			return s.DeleteWikiPage(args[0])
		},
	}

	wikiCmd.AddCommand(addCmd, lsCmd, viewCmd, editCmd, rmCmd)
	Root.AddCommand(wikiCmd)
}

func init() {
	var indexKind, indexStatus string

	indexCmd := &cobra.Command{
		Use:   "index",
		Short: "Print the wiki catalog as markdown",
		Long: "Renders the llm-wiki index: every page grouped by kind, each " +
			"with its one-line summary. Read this before drilling into pages.",
		RunE: func(cmd *cobra.Command, args []string) error {
			s, err := OpenStore()
			if err != nil {
				return err
			}
			defer s.Close()

			pages, err := s.ListWikiPages(indexKind, indexStatus, ProjectFlag(), LimitFlag())
			if err != nil {
				return err
			}
			if JSONMode() {
				return render.JSON(os.Stdout, pages)
			}
			render.WikiIndex(os.Stdout, pages)
			return nil
		},
	}

	indexCmd.Flags().StringVar(&indexKind, "kind", "", "restrict the index to one kind")
	indexCmd.Flags().StringVar(&indexStatus, "status", "", "restrict the index to one status")

	wikiCmd.AddCommand(indexCmd)
}
