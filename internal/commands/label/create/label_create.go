package create

import (
	"github.com/MakeNowJust/heredoc/v2"
	"github.com/spf13/cobra"

	gitlab "gitlab.com/gitlab-org/api/client-go"

	"gitlab.com/gitlab-org/cli/internal/cmdutils"
	"gitlab.com/gitlab-org/cli/internal/mcpannotations"
)

func NewCmdCreate(f cmdutils.Factory) *cobra.Command {
	labelCreateCmd := &cobra.Command{
		Use:     "create [flags]",
		Short:   `Create labels for a repository or project.`,
		Long:    ``,
		Aliases: []string{"new"},
		Example: heredoc.Doc(`
			$ glab label create
			$ glab label new
			$ glab label create -R owner/repo
		`),
		Args: cobra.ExactArgs(0),
		Annotations: map[string]string{
			mcpannotations.Destructive: "true",
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			var err error

			client, err := f.GitLabClient()
			if err != nil {
				return err
			}

			repo, err := f.BaseRepo()
			if err != nil {
				return err
			}

			l := &gitlab.CreateLabelOptions{}

			if s, _ := cmd.Flags().GetString("name"); s != "" {
				l.Name = gitlab.Ptr(s)
			}

			if s, _ := cmd.Flags().GetString("color"); s != "" {
				l.Color = gitlab.Ptr(s)
			}
			if s, _ := cmd.Flags().GetString("description"); s != "" {
				l.Description = gitlab.Ptr(s)
			}
			if cmd.Flags().Changed("priority") {
				if s, err := cmd.Flags().GetInt("priority"); err == nil {
					l.Priority = gitlab.Ptr(int64(s))
				} else {
					return err
				}
			}
			label, _, err := client.Labels.CreateLabel(repo.FullName(), l)
			if err != nil {
				return err
			}

			f.IO().LogInfof("Created label: %s\nWith color: %s\n", label.Name, label.Color)

			return nil
		},
	}
	labelCreateCmd.Flags().StringP("name", "n", "", "Name of the label.")
	_ = labelCreateCmd.MarkFlagRequired("name")
	labelCreateCmd.Flags().StringP("color", "c", "#428BCA", "Color of the label, in plain or HEX code.")
	labelCreateCmd.Flags().StringP("description", "d", "", "Label description.")
	labelCreateCmd.Flags().IntP("priority", "p", 0, "Label priority.")

	return labelCreateCmd
}
