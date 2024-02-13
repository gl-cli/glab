package config

import (
	ConfigCompileCmd "gitlab.com/gitlab-org/cli/commands/ci/config/compile"
	"gitlab.com/gitlab-org/cli/commands/cmdutils"

	"github.com/spf13/cobra"
)

func NewCmdConfig(f *cmdutils.Factory) *cobra.Command {
	ConfigCmd := &cobra.Command{
		Use:   "config <command> [flags]",
		Short: `Work with GitLab CI/CD configuration.`,
		Long:  ``,
	}
	ConfigCmd.AddCommand(ConfigCompileCmd.NewCmdConfigCompile(f))
	return ConfigCmd
}
