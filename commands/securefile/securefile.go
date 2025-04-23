package securefile

import (
	"github.com/MakeNowJust/heredoc/v2"
	"github.com/spf13/cobra"
	"gitlab.com/gitlab-org/cli/commands/cmdutils"

	securefileCreateCmd "gitlab.com/gitlab-org/cli/commands/securefile/create"
	securefileDownloadCmd "gitlab.com/gitlab-org/cli/commands/securefile/download"
	securefileListCmd "gitlab.com/gitlab-org/cli/commands/securefile/list"
	securefileRemoveCmd "gitlab.com/gitlab-org/cli/commands/securefile/remove"
)

func NewCmdSecurefile(f *cmdutils.Factory) *cobra.Command {
	securefileCmd := &cobra.Command{
		Use:   "securefile <command> [flags]",
		Short: `Manage secure files for a project.`,
		Long: heredoc.Docf(`
		Store up to 100 files for secure use in CI/CD pipelines. Secure files are
		stored outside of your project's repository, not in version control.
		It is safe to store sensitive information in these files. Both plain text
		and binary files are supported, but they must be smaller than 5 MB.
		`),
	}

	cmdutils.EnableRepoOverride(securefileCmd, f)

	securefileCmd.AddCommand(securefileCreateCmd.NewCmdCreate(f))
	securefileCmd.AddCommand(securefileDownloadCmd.NewCmdDownload(f))
	securefileCmd.AddCommand(securefileListCmd.NewCmdList(f))
	securefileCmd.AddCommand(securefileRemoveCmd.NewCmdRemove(f))
	return securefileCmd
}
