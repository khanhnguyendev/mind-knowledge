package cli

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/khanhnguyendev/mind-knowledge/internal/doctor"
	"github.com/khanhnguyendev/mind-knowledge/internal/render"
)

func init() {
	var scopes []string

	doctorCmd := &cobra.Command{
		Use:   "doctor",
		Short: "Report drift across the wiki, stories, and projects",
		Long: "Reports problems and repairs nothing. Exits 0 even when it " +
			"finds something: the findings are information for a skill to act on.",
		RunE: func(cmd *cobra.Command, args []string) error {
			s, err := OpenStore()
			if err != nil {
				return err
			}
			defer s.Close()

			findings, err := doctor.Run(s, scopes)
			if err != nil {
				return invalidf("%v", err)
			}
			if JSONMode() {
				return render.JSON(os.Stdout, findings)
			}

			if len(findings) == 0 {
				fmt.Println("no findings")
				return nil
			}
			rows := make([][]string, 0, len(findings))
			for _, f := range findings {
				rows = append(rows, []string{f.Check, f.ID, f.Message})
			}
			render.Table(os.Stdout, []string{"CHECK", "ID", "DETAIL"}, rows)
			return nil
		},
	}

	doctorCmd.Flags().StringSliceVar(&scopes, "scope", nil,
		"restrict to wiki, stories, or projects; repeatable")

	Root.AddCommand(doctorCmd)
}
