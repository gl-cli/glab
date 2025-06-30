package config

import (
	"gitlab.com/gitlab-org/cli/internal/cmdutils"
	ConfigCompileCmd "gitlab.com/gitlab-org/cli/internal/commands/ci/config/compile"

	"github.com/spf13/cobra"
)

func NewCmdConfig(f cmdutils.Factory) *cobra.Command {
	ConfigCmd := &cobra.Command{
		Use:   "config <command> [flags]",
		Short: `Work with GitLab CI/CD configuration.`,
		Long:  ``,
	}
	ConfigCmd.AddCommand(ConfigCompileCmd.NewCmdConfigCompile(f))
	return ConfigCmd
}
