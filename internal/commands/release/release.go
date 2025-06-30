package release

import (
	"github.com/spf13/cobra"
	"gitlab.com/gitlab-org/cli/internal/cmdutils"
	releaseCreateCmd "gitlab.com/gitlab-org/cli/internal/commands/release/create"
	releaseDeleteCmd "gitlab.com/gitlab-org/cli/internal/commands/release/delete"
	releaseDownloadCmd "gitlab.com/gitlab-org/cli/internal/commands/release/download"
	releaseListCmd "gitlab.com/gitlab-org/cli/internal/commands/release/list"
	releaseUploadCmd "gitlab.com/gitlab-org/cli/internal/commands/release/upload"
	releaseViewCmd "gitlab.com/gitlab-org/cli/internal/commands/release/view"
)

func NewCmdRelease(f cmdutils.Factory) *cobra.Command {
	releaseCmd := &cobra.Command{
		Use:   "release <command> [flags]",
		Short: `Manage GitLab releases.`,
		Long:  ``,
	}

	cmdutils.EnableRepoOverride(releaseCmd, f)

	releaseCmd.AddCommand(releaseListCmd.NewCmdReleaseList(f))
	releaseCmd.AddCommand(releaseCreateCmd.NewCmdCreate(f))
	releaseCmd.AddCommand(releaseUploadCmd.NewCmdUpload(f))
	releaseCmd.AddCommand(releaseDeleteCmd.NewCmdDelete(f))
	releaseCmd.AddCommand(releaseViewCmd.NewCmdView(f))
	releaseCmd.AddCommand(releaseDownloadCmd.NewCmdDownload(f))

	return releaseCmd
}
