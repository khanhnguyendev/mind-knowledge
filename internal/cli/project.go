package cli

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/khanhnguyendev/mind-knowledge/internal/render"
	"github.com/khanhnguyendev/mind-knowledge/internal/store"
)

func init() {
	projectCmd := &cobra.Command{
		Use:   "project",
		Short: "Register and inspect projects",
		Args:  rejectUnknownSubcommand,
		RunE:  showHelp,
	}

	// add
	var addName, addPath, addRemote string
	addCmd := &cobra.Command{
		Use:   "add",
		Short: "Register a repository as a project",
		RunE: func(cmd *cobra.Command, args []string) error {
			s, err := OpenStore()
			if err != nil {
				return err
			}
			defer s.Close()

			p, err := s.CreateProject(addName, addPath, addRemote)
			if err != nil {
				return err
			}
			return emitCreated(p, p.ID)
		},
	}
	addCmd.Flags().StringVar(&addName, "name", "", "project name (required, unique)")
	addCmd.Flags().StringVar(&addPath, "path", "", "absolute repository path (required)")
	addCmd.Flags().StringVar(&addRemote, "remote", "", "git remote URL")
	addCmd.MarkFlagRequired("name")
	addCmd.MarkFlagRequired("path")

	// ls
	var lsStatus string
	lsCmd := &cobra.Command{
		Use:   "ls",
		Short: "List projects",
		RunE: func(cmd *cobra.Command, args []string) error {
			s, err := OpenStore()
			if err != nil {
				return err
			}
			defer s.Close()

			projects, err := s.ListProjects(lsStatus, LimitFlag())
			if err != nil {
				return err
			}
			if JSONMode() {
				return render.JSON(os.Stdout, projects)
			}
			rows := make([][]string, 0, len(projects))
			for _, p := range projects {
				rows = append(rows, []string{p.ID, p.Name, p.Status, p.RepoPath})
			}
			render.Table(os.Stdout, []string{"ID", "NAME", "STATUS", "PATH"}, rows)
			return nil
		},
	}
	lsCmd.Flags().StringVar(&lsStatus, "status", "", "filter by status")

	// view
	viewCmd := &cobra.Command{
		Use:   "view <id-or-name>",
		Short: "Show one project",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			s, err := OpenStore()
			if err != nil {
				return err
			}
			defer s.Close()

			p, err := s.GetProject(args[0])
			if err != nil {
				return err
			}
			if JSONMode() {
				return render.JSON(os.Stdout, p)
			}
			fmt.Printf("%s  %s  [%s]\n%s\n", p.ID, p.Name, p.Status, p.RepoPath)
			if p.GitRemote != "" {
				fmt.Printf("remote: %s\n", p.GitRemote)
			}
			return nil
		},
	}

	// edit
	var editName, editPath, editRemote, editStatus string
	editCmd := &cobra.Command{
		Use:   "edit <id-or-name>",
		Short: "Change project fields",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			s, err := OpenStore()
			if err != nil {
				return err
			}
			defer s.Close()

			f := store.ProjectFields{}
			if cmd.Flags().Changed("name") {
				f.Name = &editName
			}
			if cmd.Flags().Changed("path") {
				f.RepoPath = &editPath
			}
			if cmd.Flags().Changed("remote") {
				f.GitRemote = &editRemote
			}
			if cmd.Flags().Changed("status") {
				f.Status = &editStatus
			}

			p, err := s.GetProject(args[0])
			if err != nil {
				return err
			}
			updated, err := s.UpdateProject(p.ID, f)
			if err != nil {
				return err
			}
			if JSONMode() {
				return render.JSON(os.Stdout, updated)
			}
			fmt.Println(updated.ID)
			return nil
		},
	}
	editCmd.Flags().StringVar(&editName, "name", "", "new name")
	editCmd.Flags().StringVar(&editPath, "path", "", "new repository path")
	editCmd.Flags().StringVar(&editRemote, "remote", "", "new git remote")
	editCmd.Flags().StringVar(&editStatus, "status", "",
		"new status: active, paused, or archived")

	// rm
	rmCmd := &cobra.Command{
		Use:   "rm <id-or-name>",
		Short: "Remove a project and everything under it",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			s, err := OpenStore()
			if err != nil {
				return err
			}
			defer s.Close()

			return s.DeleteProject(args[0])
		},
	}

	projectCmd.AddCommand(addCmd, lsCmd, viewCmd, editCmd, rmCmd)
	Root.AddCommand(projectCmd)
}

// emitCreated prints a created record: the full object under --json, the
// bare id otherwise. Every create command uses it so the contract holds.
func emitCreated(record any, id string) error {
	if JSONMode() {
		return render.JSON(os.Stdout, record)
	}
	fmt.Println(id)
	return nil
}
