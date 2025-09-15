package artifact

import (
	"github.com/MakeNowJust/heredoc/v2"
	"github.com/spf13/cobra"
	"gitlab.com/gitlab-org/cli/internal/cmdutils"
	jobArtifact "gitlab.com/gitlab-org/cli/internal/commands/job/artifact"
	"gitlab.com/gitlab-org/cli/internal/mcpannotations"
)

func NewCmdRun(f cmdutils.Factory) *cobra.Command {
	jobArtifactCmd := &cobra.Command{
		Use:     "artifact <refName> <jobName> [flags]",
		Short:   `Download all artifacts from the last pipeline.`,
		Aliases: []string{"push"},
		Example: heredoc.Doc(`
			# Download all artifacts from the main branch and build job
			$ glab ci artifact main build
			$ glab ci artifact main deploy --path="artifacts/"
		`),
		Long: ``,
		Args: cobra.ExactArgs(2),
		Annotations: map[string]string{
			mcpannotations.Safe: "true",
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			repo, err := f.BaseRepo()
			if err != nil {
				return err
			}
			client, err := f.GitLabClient()
			if err != nil {
				return err
			}
			path, err := cmd.Flags().GetString("path")
			if err != nil {
				return err
			}

			return jobArtifact.DownloadArtifacts(client, repo, path, false, args[0], args[1])
		},
		Deprecated: "use 'glab job artifact' instead.",
	}
	jobArtifactCmd.Flags().StringP("path", "p", "./", "Path to download the artifact files.")

	return jobArtifactCmd
}
