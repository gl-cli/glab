package job

import (
	"gitlab.com/gitlab-org/cli/commands/cmdutils"
	"gitlab.com/gitlab-org/cli/commands/job/artifact"

	"github.com/spf13/cobra"
)

func NewCmdJob(f *cmdutils.Factory) *cobra.Command {
	jobCmd := &cobra.Command{
		Use:   "job <command> [flags]",
		Short: `Work with GitLab CI/CD jobs.`,
		Long:  ``,
	}

	cmdutils.EnableRepoOverride(jobCmd, f)
	jobCmd.AddCommand(artifact.NewCmdArtifact(f))
	return jobCmd
}
