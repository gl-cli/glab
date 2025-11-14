package job

import (
	"github.com/spf13/cobra"

	"gitlab.com/gitlab-org/cli/internal/cmdutils"
	"gitlab.com/gitlab-org/cli/internal/commands/job/artifact"
)

func NewCmdJob(f cmdutils.Factory) *cobra.Command {
	jobCmd := &cobra.Command{
		Use:   "job <command> [flags]",
		Short: `Work with GitLab CI/CD jobs.`,
		Long:  ``,
	}

	cmdutils.EnableRepoOverride(jobCmd, f)
	jobCmd.AddCommand(artifact.NewCmdArtifact(f))
	return jobCmd
}
