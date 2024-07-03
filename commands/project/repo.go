package project

import (
	"gitlab.com/gitlab-org/cli/commands/cmdutils"
	repoCmdArchive "gitlab.com/gitlab-org/cli/commands/project/archive"
	repoCmdClone "gitlab.com/gitlab-org/cli/commands/project/clone"
	repoCmdContributors "gitlab.com/gitlab-org/cli/commands/project/contributors"
	repoCmdCreate "gitlab.com/gitlab-org/cli/commands/project/create"
	repoCmdDelete "gitlab.com/gitlab-org/cli/commands/project/delete"
	repoCmdFork "gitlab.com/gitlab-org/cli/commands/project/fork"
	repoCmdList "gitlab.com/gitlab-org/cli/commands/project/list"
	repoCmdMirror "gitlab.com/gitlab-org/cli/commands/project/mirror"
	repoCmdSearch "gitlab.com/gitlab-org/cli/commands/project/search"
	repoCmdTransfer "gitlab.com/gitlab-org/cli/commands/project/transfer"
	repoCmdView "gitlab.com/gitlab-org/cli/commands/project/view"

	"github.com/spf13/cobra"
)

func NewCmdRepo(f *cmdutils.Factory) *cobra.Command {
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
	repoCmd.AddCommand(repoCmdFork.NewCmdFork(f, nil))
	repoCmd.AddCommand(repoCmdSearch.NewCmdSearch(f))
	repoCmd.AddCommand(repoCmdTransfer.NewCmdTransfer(f))
	repoCmd.AddCommand(repoCmdView.NewCmdView(f))
	repoCmd.AddCommand(repoCmdMirror.NewCmdMirror(f))

	return repoCmd
}
