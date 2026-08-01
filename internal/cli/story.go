package cli

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/khanhnguyendev/mind-knowledge/internal/render"
	"github.com/khanhnguyendev/mind-knowledge/internal/store"
)

func init() {
	storyCmd := &cobra.Command{
		Use:   "story",
		Short: "Create, edit, and move stories",
	}

	// create
	var createEpic, createTitle, createDesc, createAcceptance string
	createCmd := &cobra.Command{
		Use:   "create",
		Short: "Create a story under an epic",
		RunE: func(cmd *cobra.Command, args []string) error {
			s, err := OpenStore()
			if err != nil {
				return err
			}
			defer s.Close()

			st, err := s.CreateStory(createEpic, createTitle, createDesc)
			if err != nil {
				return err
			}
			if createAcceptance != "" {
				st, err = s.UpdateStory(st.ID, store.StoryFields{Acceptance: &createAcceptance})
				if err != nil {
					return err
				}
			}
			return emitCreated(st, st.ID)
		},
	}
	createCmd.Flags().StringVar(&createEpic, "epic", "", "epic id (required)")
	createCmd.Flags().StringVar(&createTitle, "title", "", "story title (required)")
	createCmd.Flags().StringVarP(&createDesc, "description", "d", "", "story description")
	createCmd.Flags().StringVar(&createAcceptance, "acceptance", "",
		"acceptance criteria as a markdown checklist")
	createCmd.MarkFlagRequired("epic")
	createCmd.MarkFlagRequired("title")

	// ls
	var lsEpic, lsStatus, lsPriority string
	lsCmd := &cobra.Command{
		Use:   "ls",
		Short: "List stories",
		RunE: func(cmd *cobra.Command, args []string) error {
			s, err := OpenStore()
			if err != nil {
				return err
			}
			defer s.Close()

			stories, err := s.ListStories(store.StoryFilter{
				ProjectID: ProjectFlag(),
				EpicID:    lsEpic,
				Status:    lsStatus,
				Priority:  lsPriority,
				Limit:     LimitFlag(),
			})
			if err != nil {
				return err
			}
			if JSONMode() {
				return render.JSON(os.Stdout, stories)
			}
			rows := make([][]string, 0, len(stories))
			for _, st := range stories {
				rows = append(rows, []string{st.ID, st.Status, st.Priority, st.Title})
			}
			render.Table(os.Stdout, []string{"ID", "STATUS", "PRI", "TITLE"}, rows)
			return nil
		},
	}
	lsCmd.Flags().StringVar(&lsEpic, "epic", "", "filter by epic id")
	lsCmd.Flags().StringVar(&lsStatus, "status", "", "filter by status")
	lsCmd.Flags().StringVar(&lsPriority, "priority", "", "filter by priority")

	// view
	viewCmd := &cobra.Command{
		Use:   "view <id>",
		Short: "Show one story",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			s, err := OpenStore()
			if err != nil {
				return err
			}
			defer s.Close()

			st, err := s.GetStory(args[0])
			if err != nil {
				return err
			}
			if JSONMode() {
				return render.JSON(os.Stdout, st)
			}
			fmt.Printf("%s  [%s/%s]  %s\n", st.ID, st.Status, st.Priority, st.Title)
			// An ordered slice, not a map: ranging over a map randomizes
			// section order between runs, and plain `story view` is a
			// surface skills parse.
			for _, section := range []struct{ label, body string }{
				{"description", st.Description},
				{"acceptance", st.Acceptance},
				{"plan", st.Plan},
				{"notes", st.Notes},
			} {
				if section.body != "" {
					fmt.Printf("\n## %s\n%s\n", section.label, section.body)
				}
			}
			return nil
		},
	}

	// edit
	var (
		editTitle, editDesc, editAcceptance string
		editPlan, editNotes, editAppend     string
		editPriority, editEpic              string
	)
	editCmd := &cobra.Command{
		Use:   "edit <id>",
		Short: "Change story fields",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			s, err := OpenStore()
			if err != nil {
				return err
			}
			defer s.Close()

			f := store.StoryFields{}
			if cmd.Flags().Changed("title") {
				f.Title = &editTitle
			}
			if cmd.Flags().Changed("description") {
				f.Description = &editDesc
			}
			if cmd.Flags().Changed("acceptance") {
				f.Acceptance = &editAcceptance
			}
			if cmd.Flags().Changed("plan") {
				f.Plan = &editPlan
			}
			if cmd.Flags().Changed("notes") {
				f.Notes = &editNotes
			}
			if cmd.Flags().Changed("append-notes") {
				f.AppendNotes = &editAppend
			}
			if cmd.Flags().Changed("priority") {
				f.Priority = &editPriority
			}
			if cmd.Flags().Changed("epic") {
				f.EpicID = &editEpic
			}

			st, err := s.UpdateStory(args[0], f)
			if err != nil {
				return err
			}
			return emitCreated(st, st.ID)
		},
	}
	editCmd.Flags().StringVar(&editTitle, "title", "", "new title")
	editCmd.Flags().StringVarP(&editDesc, "description", "d", "", "new description")
	editCmd.Flags().StringVar(&editAcceptance, "acceptance", "", "new acceptance criteria")
	editCmd.Flags().StringVar(&editPlan, "plan", "", "implementation plan")
	editCmd.Flags().StringVar(&editNotes, "notes", "", "replace notes")
	editCmd.Flags().StringVar(&editAppend, "append-notes", "", "append to notes")
	editCmd.Flags().StringVar(&editPriority, "priority", "", "new priority: low, med, or high")
	editCmd.Flags().StringVar(&editEpic, "epic", "", "move to another epic")

	// mv
	var mvTo string
	var mvPos int
	mvCmd := &cobra.Command{
		Use:   "mv <id>",
		Short: "Move a story to a new status or position",
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

			f := store.StoryFields{}
			if toChanged {
				f.Status = &mvTo
			}
			if posChanged {
				f.Position = &mvPos
			}

			st, err := s.UpdateStory(args[0], f)
			if err != nil {
				return err
			}
			return emitCreated(st, st.ID)
		},
	}
	mvCmd.Flags().StringVar(&mvTo, "to", "",
		"new status: backlog, ready, in-progress, review, done, or dropped")
	mvCmd.Flags().IntVar(&mvPos, "pos", 0, "new position among sibling stories")

	// rm
	rmCmd := &cobra.Command{
		Use:   "rm <id>",
		Short: "Remove a story",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			s, err := OpenStore()
			if err != nil {
				return err
			}
			defer s.Close()

			return s.DeleteStory(args[0])
		},
	}

	storyCmd.AddCommand(createCmd, lsCmd, viewCmd, editCmd, mvCmd, rmCmd)
	Root.AddCommand(storyCmd)
}
