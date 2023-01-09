package create

import (
	"fmt"

	"github.com/MakeNowJust/heredoc"
	"gitlab.com/gitlab-org/cli/api"

	"github.com/spf13/cobra"
	"github.com/xanzy/go-gitlab"
	"gitlab.com/gitlab-org/cli/commands/cmdutils"
)

func NewCmdCreate(f *cmdutils.Factory) *cobra.Command {
	labelCreateCmd := &cobra.Command{
		Use:     "create [flags]",
		Short:   `Create labels for repository/project`,
		Long:    ``,
		Aliases: []string{"new"},
		Example: heredoc.Doc(`
			glab label create
			glab label new
			glab label create -R owner/repo
		`),
		Args: cobra.ExactArgs(0),
		RunE: func(cmd *cobra.Command, args []string) error {
			var err error

			apiClient, err := f.HttpClient()
			if err != nil {
				return err
			}

			repo, err := f.BaseRepo()
			if err != nil {
				return err
			}

			l := &gitlab.CreateLabelOptions{}

			if s, _ := cmd.Flags().GetString("name"); s != "" {
				l.Name = gitlab.String(s)
			}

			if s, _ := cmd.Flags().GetString("color"); s != "" {
				l.Color = gitlab.String(s)
			}
			if s, _ := cmd.Flags().GetString("description"); s != "" {
				l.Description = gitlab.String(s)
			}
			label, err := api.CreateLabel(apiClient, repo.FullName(), l)
			if err != nil {
				return err
			}
			fmt.Fprintf(f.IO.StdOut, "Created label: %s\nWith color: %s\n", label.Name, label.Color)

			return nil
		},
	}
	labelCreateCmd.Flags().StringP("name", "n", "", "Name of label")
	_ = labelCreateCmd.MarkFlagRequired("name")
	labelCreateCmd.Flags().StringP("color", "c", "#428BCA", "Color of label in plain or HEX code.")
	labelCreateCmd.Flags().StringP("description", "d", "", "Label description")

	return labelCreateCmd
}
