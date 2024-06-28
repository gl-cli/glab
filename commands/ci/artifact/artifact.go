package ci

import (
	"github.com/MakeNowJust/heredoc/v2"
	"github.com/spf13/cobra"
	"gitlab.com/gitlab-org/cli/commands/cmdutils"
	jobArtifact "gitlab.com/gitlab-org/cli/commands/job/artifact"
)

func NewCmdRun(f *cmdutils.Factory) *cobra.Command {
	jobArtifactCmd := &cobra.Command{
		Use:     "artifact <refName> <jobName> [flags]",
		Short:   `Download all artifacts from the last pipeline.`,
		Aliases: []string{"push"},
		Example: heredoc.Doc(`
	glab ci artifact main build
	glab ci artifact main deploy --path="artifacts/"
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

			return jobArtifact.DownloadArtifacts(apiClient, repo, path, args[0], args[1])
		},
		Deprecated: "use 'glab job artifact' instead.",
	}
	jobArtifactCmd.Flags().StringP("path", "p", "./", "Path to download the artifact files.")

	return jobArtifactCmd
}
