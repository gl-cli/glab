package config

import (
	"fmt"

	"github.com/MakeNowJust/heredoc/v2"
	"github.com/spf13/cobra"

	"gitlab.com/gitlab-org/cli/commands/cmdutils"
)

var isGlobal bool

func NewCmdConfig(f *cmdutils.Factory) *cobra.Command {
	configCmd := &cobra.Command{
		Use:   "config [flags]",
		Short: `Set and get glab settings.`,
		Long: heredoc.Docf(`Get and set key/value strings.

Current respected settings:

- token: Your GitLab access token. Defaults to environment variables.
- host: If unset, defaults to %[1]shttps://gitlab.com%[1]s.
- browser: If unset, uses the default browser. Override with environment variable $BROWSER.
- editor: If unset, uses the default editor. Override with environment variable $EDITOR.
- visual: Takes precedence over 'editor'. If unset, uses the default editor. Override with environment variable $VISUAL.
- glamour_style: Your desired Markdown renderer style. Options are dark, light, notty. Custom styles are available using [glamour](https://github.com/charmbracelet/glamour#styles).
- glab_pager: Your desired pager command to use, such as 'less -R'.
- check_update: If true, notifies of new versions of glab. Defaults to true.
- display_hyperlinks: If true, and using a TTY, outputs hyperlinks for issues and merge request lists. Defaults to false.
`, "`"),
		Aliases: []string{"conf"},
	}

	configCmd.Flags().BoolVarP(&isGlobal, "global", "g", false, "Use global config file.")

	configCmd.AddCommand(NewCmdConfigGet(f))
	configCmd.AddCommand(NewCmdConfigSet(f))

	return configCmd
}

func NewCmdConfigGet(f *cmdutils.Factory) *cobra.Command {
	var hostname string

	cmd := &cobra.Command{
		Use:   "get <key>",
		Short: "Prints the value of a given configuration key.",
		Long:  ``,
		Example: `
  $ glab config get editor
  vim
  $ glab config get glamour_style
  notty
`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := f.Config()
			if err != nil {
				return err
			}

			val, err := cfg.Get(hostname, args[0])
			if err != nil {
				return err
			}

			if val != "" {
				fmt.Fprintf(f.IO.StdOut, "%s\n", val)
			}
			return nil
		},
	}

	cmd.Flags().StringVarP(&hostname, "host", "h", "", "Get per-host setting.")
	cmd.Flags().BoolP("global", "g", false, "Read from global config file (~/.config/glab-cli/config.yml). Default: checks 'Environment variables → Local → Global'.")

	return cmd
}

func NewCmdConfigSet(f *cmdutils.Factory) *cobra.Command {
	var hostname string

	cmd := &cobra.Command{
		Use:   "set <key> <value>",
		Short: "Updates configuration with the value of a given key.",
		Long: `Update the configuration by setting a key to a value.
Use 'glab config set --global' to set a global config.
Specifying the '--hostname' flag also saves in the global configuration file.
`,
		Example: `
  glab config set editor vim
  glab config set token xxxxx -h gitlab.com
  glab config set check_update false --global
`,
		Args: cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := f.Config()
			if err != nil {
				return err
			}

			localCfg, _ := cfg.Local()

			key, value := args[0], args[1]
			if isGlobal || hostname != "" {
				err = cfg.Set(hostname, key, value)
			} else {
				err = localCfg.Set(key, value)
			}

			if err != nil {
				return fmt.Errorf("failed to set %q to %q: %w", key, value, err)
			}

			if isGlobal || hostname != "" {
				err = cfg.Write()
			} else {
				err = localCfg.Write()
			}

			if err != nil {
				return fmt.Errorf("failed to write configuration to disk: %w", err)
			}
			return nil
		},
	}

	cmd.Flags().StringVarP(&hostname, "host", "h", "", "Set per-host setting.")
	cmd.Flags().BoolVarP(&isGlobal, "global", "g", false, "Write to global '~/.config/glab-cli/config.yml' file rather than the repository's '.git/glab-cli/config.yml' file.")
	return cmd
}
