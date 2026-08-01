package cli

import (
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/khanhnguyendev/mind-knowledge/internal/render"
)

func init() {
	linkCmd := &cobra.Command{
		Use:   "link",
		Short: "Connect entities to each other",
		Long: "Endpoints are written kind:reference, for example " +
			"wiki:auth-model or story:b4g3l2. Valid kinds are project, epic, " +
			"story, source, and wiki. An edge may join entities in different " +
			"projects, so no link command takes -p/--project.",
		PersistentPreRunE: crossProjectPreRun("links"),
	}

	// add
	var addFrom, addTo, addRelation string
	addCmd := &cobra.Command{
		Use:   "add",
		Short: "Add an edge between two entities",
		RunE: func(cmd *cobra.Command, args []string) error {
			fromKind, fromRef, err := parseEntityRef(addFrom)
			if err != nil {
				return err
			}
			toKind, toRef, err := parseEntityRef(addTo)
			if err != nil {
				return err
			}

			s, err := OpenStore()
			if err != nil {
				return err
			}
			defer s.Close()

			l, err := s.AddLink(fromKind, fromRef, toKind, toRef, addRelation)
			if err != nil {
				return err
			}
			if JSONMode() {
				return render.JSON(os.Stdout, l)
			}
			return nil
		},
	}
	addCmd.Flags().StringVar(&addFrom, "from", "", "source endpoint as kind:reference (required)")
	addCmd.Flags().StringVar(&addTo, "to", "", "target endpoint as kind:reference (required)")
	addCmd.Flags().StringVar(&addRelation, "relation", "",
		"derived-from, supersedes, references, or implements (required)")
	addCmd.MarkFlagRequired("from")
	addCmd.MarkFlagRequired("to")
	addCmd.MarkFlagRequired("relation")

	// ls
	var lsFrom, lsTo, lsRelation string
	lsCmd := &cobra.Command{
		Use:   "ls",
		Short: "List edges",
		RunE: func(cmd *cobra.Command, args []string) error {
			s, err := OpenStore()
			if err != nil {
				return err
			}
			defer s.Close()

			var fromKind, fromID, toKind, toID string
			if lsFrom != "" {
				kind, ref, err := parseEntityRef(lsFrom)
				if err != nil {
					return err
				}
				resolved, err := s.ResolveEntity(kind, ref)
				if err != nil {
					return err
				}
				fromKind, fromID = kind, resolved
			}
			if lsTo != "" {
				kind, ref, err := parseEntityRef(lsTo)
				if err != nil {
					return err
				}
				resolved, err := s.ResolveEntity(kind, ref)
				if err != nil {
					return err
				}
				toKind, toID = kind, resolved
			}

			links, err := s.ListLinks(fromKind, fromID, toKind, toID, lsRelation)
			if err != nil {
				return err
			}
			if JSONMode() {
				return render.JSON(os.Stdout, links)
			}
			rows := make([][]string, 0, len(links))
			for _, l := range links {
				rows = append(rows, []string{
					l.FromKind + ":" + l.FromID,
					l.Relation,
					l.ToKind + ":" + l.ToID,
				})
			}
			render.Table(os.Stdout, []string{"FROM", "RELATION", "TO"}, rows)
			return nil
		},
	}
	lsCmd.Flags().StringVar(&lsFrom, "from", "", "filter by source endpoint")
	lsCmd.Flags().StringVar(&lsTo, "to", "", "filter by target endpoint")
	lsCmd.Flags().StringVar(&lsRelation, "relation", "", "filter by relation")

	// rm
	var rmFrom, rmTo, rmRelation string
	rmCmd := &cobra.Command{
		Use:   "rm",
		Short: "Remove an edge",
		RunE: func(cmd *cobra.Command, args []string) error {
			fromKind, fromRef, err := parseEntityRef(rmFrom)
			if err != nil {
				return err
			}
			toKind, toRef, err := parseEntityRef(rmTo)
			if err != nil {
				return err
			}

			s, err := OpenStore()
			if err != nil {
				return err
			}
			defer s.Close()

			return s.RemoveLink(fromKind, fromRef, toKind, toRef, rmRelation)
		},
	}
	rmCmd.Flags().StringVar(&rmFrom, "from", "", "source endpoint (required)")
	rmCmd.Flags().StringVar(&rmTo, "to", "", "target endpoint (required)")
	rmCmd.Flags().StringVar(&rmRelation, "relation", "", "relation (required)")
	rmCmd.MarkFlagRequired("from")
	rmCmd.MarkFlagRequired("to")
	rmCmd.MarkFlagRequired("relation")

	linkCmd.AddCommand(addCmd, lsCmd, rmCmd)
	Root.AddCommand(linkCmd)
}

// parseEntityRef splits "wiki:auth-model" into its kind and reference.
func parseEntityRef(ref string) (kind, entityRef string, err error) {
	parts := strings.SplitN(ref, ":", 2)
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return "", "", invalidf(
			"endpoint %q must be written kind:reference, for example wiki:auth-model", ref)
	}
	return parts[0], parts[1], nil
}
