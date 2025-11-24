package artifact

import (
	"github.com/MakeNowJust/heredoc/v2"
	"github.com/spf13/cobra"

	"gitlab.com/gitlab-org/cli/internal/cmdutils"
	"gitlab.com/gitlab-org/cli/internal/mcpannotations"
)

func NewCmdArtifact(f cmdutils.Factory) *cobra.Command {
	jobArtifactCmd := &cobra.Command{
		Use:     "artifact <refName> <jobName> [flags]",
		Short:   `Download all artifacts from the last pipeline.`,
		Aliases: []string{"push"},
		Example: heredoc.Doc(`
			$ glab job artifact main build
			$ glab job artifact main deploy --path="artifacts/"
			$ glab job artifact main deploy --list-paths
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
			listPaths, err := cmd.Flags().GetBool("list-paths")
			if err != nil {
				return err
			}
			return DownloadArtifacts(client, repo, path, listPaths, args[0], args[1])
		},
	}
	jobArtifactCmd.Flags().StringP("path", "p", "./", "Path to download the artifact files.")
	jobArtifactCmd.Flags().BoolP("list-paths", "l", false, "Print the paths of downloaded artifacts.")
	return jobArtifactCmd
}
