package release

import (
	"github.com/spf13/cobra"
	"gitlab.com/gitlab-org/cli/commands/cmdutils"
	releaseCreateCmd "gitlab.com/gitlab-org/cli/commands/release/create"
	releaseDeleteCmd "gitlab.com/gitlab-org/cli/commands/release/delete"
	releaseDownloadCmd "gitlab.com/gitlab-org/cli/commands/release/download"
	releaseListCmd "gitlab.com/gitlab-org/cli/commands/release/list"
	releaseUploadCmd "gitlab.com/gitlab-org/cli/commands/release/upload"
	releaseViewCmd "gitlab.com/gitlab-org/cli/commands/release/view"
)

func NewCmdRelease(f *cmdutils.Factory) *cobra.Command {
	var releaseCmd = &cobra.Command{
		Use:   "release <command> [flags]",
		Short: `Manage GitLab releases`,
		Long:  ``,
	}

	cmdutils.EnableRepoOverride(releaseCmd, f)

	releaseCmd.AddCommand(releaseListCmd.NewCmdReleaseList(f))
	releaseCmd.AddCommand(releaseCreateCmd.NewCmdCreate(f, nil))
	releaseCmd.AddCommand(releaseUploadCmd.NewCmdUpload(f, nil))
	releaseCmd.AddCommand(releaseDeleteCmd.NewCmdDelete(f, nil))
	releaseCmd.AddCommand(releaseViewCmd.NewCmdView(f, nil))
	releaseCmd.AddCommand(releaseDownloadCmd.NewCmdDownload(f, nil))

	return releaseCmd
}
