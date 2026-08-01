package cli

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/khanhnguyendev/mind-knowledge/internal/render"
	"github.com/khanhnguyendev/mind-knowledge/internal/store"
)

func init() {
	epicCmd := &cobra.Command{
		Use:   "epic",
		Short: "Create and move epics",
	}

	// create
	var createTitle, createDesc string
	createCmd := &cobra.Command{
		Use:   "create",
		Short: "Create an epic under a project",
		RunE: func(cmd *cobra.Command, args []string) error {
			s, err := OpenStore()
			if err != nil {
				return err
			}
			defer s.Close()

			project := ProjectFlag()
			if project == "" {
				return invalidf("a project is required; pass -p/--project")
			}

			e, err := s.CreateEpic(project, createTitle, createDesc)
			if err != nil {
				return err
			}
			return emitCreated(e, e.ID)
		},
	}
	createCmd.Flags().StringVar(&createTitle, "title", "", "epic title (required)")
	createCmd.Flags().StringVarP(&createDesc, "description", "d", "", "epic description")
	createCmd.MarkFlagRequired("title")

	// ls
	var lsStatus string
	lsCmd := &cobra.Command{
		Use:   "ls",
		Short: "List epics",
		RunE: func(cmd *cobra.Command, args []string) error {
			s, err := OpenStore()
			if err != nil {
				return err
			}
			defer s.Close()

			epics, err := s.ListEpics(ProjectFlag(), lsStatus, LimitFlag())
			if err != nil {
				return err
			}
			if JSONMode() {
				return render.JSON(os.Stdout, epics)
			}
			rows := make([][]string, 0, len(epics))
			for _, e := range epics {
				rows = append(rows, []string{e.ID, e.Status, e.Title})
			}
			render.Table(os.Stdout, []string{"ID", "STATUS", "TITLE"}, rows)
			return nil
		},
	}
	lsCmd.Flags().StringVar(&lsStatus, "status", "", "filter by status")

	// view
	viewCmd := &cobra.Command{
		Use:   "view <id>",
		Short: "Show one epic",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			s, err := OpenStore()
			if err != nil {
				return err
			}
			defer s.Close()

			e, err := s.GetEpic(args[0])
			if err != nil {
				return err
			}
			if JSONMode() {
				return render.JSON(os.Stdout, e)
			}
			fmt.Printf("%s  [%s]  %s\n", e.ID, e.Status, e.Title)
			if e.Description != "" {
				fmt.Printf("\n%s\n", e.Description)
			}
			return nil
		},
	}

	// edit
	var editTitle, editDesc, editSetProject string
	editCmd := &cobra.Command{
		Use:   "edit <id>",
		Short: "Change epic fields",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			s, err := OpenStore()
			if err != nil {
				return err
			}
			defer s.Close()

			f := store.EpicFields{}
			if cmd.Flags().Changed("title") {
				f.Title = &editTitle
			}
			if cmd.Flags().Changed("description") {
				f.Description = &editDesc
			}
			// Reassignment reads --set-project, never -p. Overloading
			// -p here meant that a skill threading -p through every
			// call — which the README tells it to do — silently moved
			// the epic to that project on every edit.
			if cmd.Flags().Changed("set-project") {
				f.ProjectID = &editSetProject
			}

			e, err := s.UpdateEpic(args[0], f)
			if err != nil {
				return err
			}
			return emitCreated(e, e.ID)
		},
	}
	editCmd.Flags().StringVar(&editTitle, "title", "", "new title")
	editCmd.Flags().StringVarP(&editDesc, "description", "d", "", "new description")
	editCmd.Flags().StringVar(&editSetProject, "set-project", "",
		"move the epic to this project (id or name)")

	// mv
	var mvTo string
	var mvPos int
	mvCmd := &cobra.Command{
		Use:   "mv <id>",
		Short: "Move an epic to a new status or position",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			toChanged := cmd.Flags().Changed("to")
			posChanged := cmd.Flags().Changed("pos")
			if !toChanged && !posChanged {
				return invalidf("mv needs --to <status> or --pos <n>")
			}

			s, err := OpenStore()
			if err != nil {
				return err
			}
			defer s.Close()

			f := store.EpicFields{}
			if toChanged {
				f.Status = &mvTo
			}
			if posChanged {
				f.Position = &mvPos
			}

			e, err := s.UpdateEpic(args[0], f)
			if err != nil {
				return err
			}
			return emitCreated(e, e.ID)
		},
	}
	mvCmd.Flags().StringVar(&mvTo, "to", "",
		"new status: backlog, in-progress, done, or dropped")
	mvCmd.Flags().IntVar(&mvPos, "pos", 0, "new position among sibling epics")

	// rm
	rmCmd := &cobra.Command{
		Use:   "rm <id>",
		Short: "Remove an epic and its stories",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			s, err := OpenStore()
			if err != nil {
				return err
			}
			defer s.Close()

			return s.DeleteEpic(args[0])
		},
	}

	epicCmd.AddCommand(createCmd, lsCmd, viewCmd, editCmd, mvCmd, rmCmd)
	Root.AddCommand(epicCmd)
}
