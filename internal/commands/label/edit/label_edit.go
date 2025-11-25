package edit

import (
	"fmt"

	"github.com/MakeNowJust/heredoc/v2"
	"github.com/spf13/cobra"

	gitlab "gitlab.com/gitlab-org/api/client-go"

	"gitlab.com/gitlab-org/cli/internal/cmdutils"
	"gitlab.com/gitlab-org/cli/internal/mcpannotations"
)

func NewCmdEdit(f cmdutils.Factory) *cobra.Command {
	var labelID int

	LabelUpdateCmd := &cobra.Command{
		Use:   "edit [flags]",
		Short: `Edit group or project label.`,
		Long:  ``,
		Example: heredoc.Doc(`
			$ glab label edit
			$ glab label edit -R owner/repo
		`),
		Args: cobra.ExactArgs(0),
		Annotations: map[string]string{
			mcpannotations.Destructive: "true",
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			var err error
			var change string

			client, err := f.GitLabClient()
			if err != nil {
				return err
			}

			repo, err := f.BaseRepo()
			if err != nil {
				return err
			}

			l := &gitlab.UpdateLabelOptions{}

			if s, _ := cmd.Flags().GetString("new-name"); s != "" {
				l.Name = gitlab.Ptr(s)
				change += fmt.Sprintf("Updated name: %s\n", s)
			}
			if s, _ := cmd.Flags().GetString("color"); s != "" {
				l.Color = gitlab.Ptr(s)
				change += fmt.Sprintf("Updated color: %s\n", s)
			}
			if s, _ := cmd.Flags().GetString("description"); s != "" {
				l.Description = gitlab.Ptr(s)
				change += fmt.Sprintf("Updated description: %s\n", s)
			}
			if cmd.Flags().Changed("priority") {
				if s, err := cmd.Flags().GetInt("priority"); err == nil {
					l.Priority = gitlab.Ptr(int64(s))
					change += fmt.Sprintf("Updated priority: %d\n", s)
				} else {
					return err
				}
			}

			label, _, err := client.Labels.UpdateLabel(repo.FullName(), labelID, l)
			if err != nil {
				return err
			}

			f.IO().LogInfof("Updating \"%s\" label\n%s", label.Name, change)

			return nil
		},
	}

	LabelUpdateCmd.Flags().IntVarP(&labelID, "label-id", "l", 0, "The label ID we are updating.")
	_ = LabelUpdateCmd.MarkFlagRequired("label-id")

	LabelUpdateCmd.Flags().StringP("new-name", "n", "", "The new name of the label.")
	LabelUpdateCmd.Flags().StringP("color", "c", "", "The color of the label given in 6-digit hex notation with leading ‘#’ sign.")
	LabelUpdateCmd.MarkFlagsOneRequired("new-name", "color")
	LabelUpdateCmd.Flags().StringP("description", "d", "", "Label description.")
	LabelUpdateCmd.Flags().IntP("priority", "p", 0, "Label priority.")

	return LabelUpdateCmd
}
