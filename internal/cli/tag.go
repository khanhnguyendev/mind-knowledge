package cli

import (
	"os"

	"github.com/spf13/cobra"

	"github.com/khanhnguyendev/mind-knowledge/internal/render"
)

func init() {
	tagCmd := &cobra.Command{
		Use:   "tag",
		Short: "Group entities with flat labels",
		Long: "Targets are written kind:reference, for example story:b4g3l2. " +
			"Tags cut across projects and epics, so no tag command takes " +
			"-p/--project.",
		PersistentPreRunE: crossProjectPreRun("tags"),
	}

	// add
	var addOn string
	addCmd := &cobra.Command{
		Use:   "add <name>",
		Short: "Attach a tag to an entity",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			kind, ref, err := parseEntityRef(addOn)
			if err != nil {
				return err
			}

			s, err := OpenStore()
			if err != nil {
				return err
			}
			defer s.Close()

			return s.AddTag(args[0], kind, ref)
		},
	}
	addCmd.Flags().StringVar(&addOn, "on", "", "target as kind:reference (required)")
	addCmd.MarkFlagRequired("on")

	// ls
	var lsOn, lsTag string
	lsCmd := &cobra.Command{
		Use:   "ls",
		Short: "List tag names, an entity's tags, or a tag's entities",
		RunE: func(cmd *cobra.Command, args []string) error {
			// Flags are mutually exclusive: --on and --tag cannot both be set
			if lsOn != "" && lsTag != "" {
				return invalidf("--on and --tag are mutually exclusive")
			}

			s, err := OpenStore()
			if err != nil {
				return err
			}
			defer s.Close()

			switch {
			case lsOn != "":
				kind, ref, err := parseEntityRef(lsOn)
				if err != nil {
					return err
				}
				tags, err := s.TagsFor(kind, ref)
				if err != nil {
					return err
				}
				if JSONMode() {
					return render.JSON(os.Stdout, tags)
				}
				for _, name := range tags {
					os.Stdout.WriteString(name + "\n")
				}
				return nil

			case lsTag != "":
				tagged, err := s.TaggedWith(lsTag)
				if err != nil {
					return err
				}
				if JSONMode() {
					return render.JSON(os.Stdout, tagged)
				}
				rows := make([][]string, 0, len(tagged))
				for _, item := range tagged {
					rows = append(rows, []string{item.FromKind, item.FromID})
				}
				render.Table(os.Stdout, []string{"KIND", "ID"}, rows)
				return nil

			default:
				names, err := s.ListTags()
				if err != nil {
					return err
				}
				if JSONMode() {
					return render.JSON(os.Stdout, names)
				}
				for _, name := range names {
					os.Stdout.WriteString(name + "\n")
				}
				return nil
			}
		},
	}
	lsCmd.Flags().StringVar(&lsOn, "on", "", "show the tags attached to this entity (cannot use with --tag)")
	lsCmd.Flags().StringVar(&lsTag, "tag", "", "show the entities carrying this tag (cannot use with --on)")

	// rm
	var rmOn string
	rmCmd := &cobra.Command{
		Use:   "rm <name>",
		Short: "Detach a tag from an entity",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			kind, ref, err := parseEntityRef(rmOn)
			if err != nil {
				return err
			}

			s, err := OpenStore()
			if err != nil {
				return err
			}
			defer s.Close()

			return s.RemoveTag(args[0], kind, ref)
		},
	}
	rmCmd.Flags().StringVar(&rmOn, "on", "", "target as kind:reference (required)")
	rmCmd.MarkFlagRequired("on")

	tagCmd.AddCommand(addCmd, lsCmd, rmCmd)
	Root.AddCommand(tagCmd)
}
