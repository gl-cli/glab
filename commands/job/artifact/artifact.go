package artifact

import (
	"github.com/MakeNowJust/heredoc/v2"
	"github.com/spf13/cobra"
	"gitlab.com/gitlab-org/cli/commands/cmdutils"
)

func NewCmdArtifact(f *cmdutils.Factory) *cobra.Command {
	jobArtifactCmd := &cobra.Command{
		Use:     "artifact <refName> <jobName> [flags]",
		Short:   `Download all artifacts from the last pipeline.`,
		Aliases: []string{"push"},
		Example: heredoc.Doc(`
	glab job artifact main build
	glab job artifact main deploy --path="artifacts/"
	`),
		Long: ``,
		Args: cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			repo, err := f.BaseRepo()
			if err != nil {
				return err
			}
			apiClient, err := f.HttpClient()
			if err != nil {
				return err
			}
			path, err := cmd.Flags().GetString("path")
			if err != nil {
				return err
			}
			return DownloadArtifacts(apiClient, repo, path, args[0], args[1])
		},
	}
	jobArtifactCmd.Flags().StringP("path", "p", "./", "Path to download the artifact files.")
	return jobArtifactCmd
}
