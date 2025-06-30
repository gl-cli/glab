package project

import (
	"gitlab.com/gitlab-org/cli/internal/cmdutils"
	repoCmdArchive "gitlab.com/gitlab-org/cli/internal/commands/project/archive"
	repoCmdClone "gitlab.com/gitlab-org/cli/internal/commands/project/clone"
	repoCmdContributors "gitlab.com/gitlab-org/cli/internal/commands/project/contributors"
	repoCmdCreate "gitlab.com/gitlab-org/cli/internal/commands/project/create"
	repoCmdDelete "gitlab.com/gitlab-org/cli/internal/commands/project/delete"
	repoCmdFork "gitlab.com/gitlab-org/cli/internal/commands/project/fork"
	repoCmdList "gitlab.com/gitlab-org/cli/internal/commands/project/list"
	repoCmdMirror "gitlab.com/gitlab-org/cli/internal/commands/project/mirror"
	repoCmdPublish "gitlab.com/gitlab-org/cli/internal/commands/project/publish"
	repoCmdSearch "gitlab.com/gitlab-org/cli/internal/commands/project/search"
	repoCmdTransfer "gitlab.com/gitlab-org/cli/internal/commands/project/transfer"
	repoCmdUpdate "gitlab.com/gitlab-org/cli/internal/commands/project/update"
	repoCmdView "gitlab.com/gitlab-org/cli/internal/commands/project/view"

	"github.com/spf13/cobra"
)

func NewCmdRepo(f cmdutils.Factory) *cobra.Command {
	repoCmd := &cobra.Command{
		Use:     "repo <command> [flags]",
		Short:   `Work with GitLab repositories and projects.`,
		Long:    ``,
		Aliases: []string{"project"},
	}

	repoCmd.AddCommand(repoCmdArchive.NewCmdArchive(f))
	repoCmd.AddCommand(repoCmdClone.NewCmdClone(f, nil))
	repoCmd.AddCommand(repoCmdContributors.NewCmdContributors(f))
	repoCmd.AddCommand(repoCmdList.NewCmdList(f))
	repoCmd.AddCommand(repoCmdCreate.NewCmdCreate(f))
	repoCmd.AddCommand(repoCmdDelete.NewCmdDelete(f))
	repoCmd.AddCommand(repoCmdFork.NewCmdFork(f))
	repoCmd.AddCommand(repoCmdSearch.NewCmdSearch(f))
	repoCmd.AddCommand(repoCmdTransfer.NewCmdTransfer(f))
	repoCmd.AddCommand(repoCmdUpdate.NewCmdUpdate(f))
	repoCmd.AddCommand(repoCmdView.NewCmdView(f))
	repoCmd.AddCommand(repoCmdMirror.NewCmdMirror(f))
	repoCmd.AddCommand(repoCmdPublish.NewCmdPublish(f))

	return repoCmd
}
