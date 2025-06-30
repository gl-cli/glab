package config

import (
	"fmt"

	"github.com/MakeNowJust/heredoc/v2"
	"github.com/spf13/cobra"

	"gitlab.com/gitlab-org/cli/internal/browser"
	"gitlab.com/gitlab-org/cli/internal/cmdutils"
	"gitlab.com/gitlab-org/cli/internal/config"
)

func NewCmdConfig(f cmdutils.Factory) *cobra.Command {
	var isGlobal bool

	configCmd := &cobra.Command{
		Use:   "config [flags]",
		Short: `Manage glab settings.`,
		Long: heredoc.Docf(`Manage key/value strings.

Current respected settings:

- browser: If unset, uses the default browser. Override with environment variable $BROWSER.
- check_update: If true, notifies of new versions of glab. Defaults to true. Override with environment variable $GLAB_CHECK_UPDATE.
- display_hyperlinks: If true, and using a TTY, outputs hyperlinks for issues and merge request lists. Defaults to false.
- editor: If unset, uses the default editor. Override with environment variable $EDITOR.
- glab_pager: Your desired pager command to use, such as 'less -R'.
- glamour_style: Your desired Markdown renderer style. Options are dark, light, notty. Custom styles are available using [glamour](https://github.com/charmbracelet/glamour#styles).
- host: If unset, defaults to %[1]shttps://gitlab.com%[1]s.
- token: Your GitLab access token. Defaults to environment variables.
- visual: Takes precedence over 'editor'. If unset, uses the default editor. Override with environment variable $VISUAL.
`, "`"),
		Aliases: []string{"conf"},
	}

	configCmd.Flags().BoolVarP(&isGlobal, "global", "g", false, "Use global config file.")

	configCmd.AddCommand(NewCmdConfigGet(f))
	configCmd.AddCommand(NewCmdConfigSet(f))
	configCmd.AddCommand(NewCmdConfigEdit(f))

	return configCmd
}

func NewCmdConfigGet(f cmdutils.Factory) *cobra.Command {
	var hostname string

	cmd := &cobra.Command{
		Use:   "get <key>",
		Short: "Prints the value of a given configuration key.",
		Long:  ``,
		Example: heredoc.Doc(`
  		$ glab config get editor
  		> vim

  		$ glab config get glamour_style
  		> notty
		`),
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg := f.Config()

			val, err := cfg.Get(hostname, args[0])
			if err != nil {
				return err
			}

			if val != "" {
				fmt.Fprintf(f.IO().StdOut, "%s\n", val)
			}
			return nil
		},
	}

	cmd.Flags().StringVarP(&hostname, "host", "h", "", "Get per-host setting.")
	cmd.Flags().BoolP("global", "g", false, "Read from global config file (~/.config/glab-cli/config.yml). (default checks 'Environment variables → Local → Global')")

	return cmd
}

func NewCmdConfigSet(f cmdutils.Factory) *cobra.Command {
	var hostname string
	var isGlobal bool

	cmd := &cobra.Command{
		Use:   "set <key> <value>",
		Short: "Updates configuration with the value of a given key.",
		Long: `Update the configuration by setting a key to a value.
Use 'glab config set --global' to set a global config.
Specifying the '--hostname' flag also saves in the global configuration file.
`,
		Example: heredoc.Doc(`
- glab config set editor vim
- glab config set token xxxxx -h gitlab.com
- glab config set check_update false --global`),
		Args: cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg := f.Config()

			localCfg, _ := cfg.Local()

			key, value := args[0], args[1]
			var err error
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

func NewCmdConfigEdit(f cmdutils.Factory) *cobra.Command {
	var isLocal bool

	cmd := &cobra.Command{
		Use:   "edit",
		Short: "Opens the glab configuration file.",
		Long: heredoc.Doc(`Opens the glab configuration file.
The command uses the following order when choosing the editor to use:

1. 'glab_editor' field in the configuration file
2. 'VISUAL' environment variable
3. 'EDITOR' environment variable
`),
		Example: heredoc.Doc(`
			Open the configuration file with the default editor
			- glab config edit

			Open the configuration file with vim
			- EDITOR=vim glab config edit

			Set vim to be used for all future 'glab config edit' invocations
			- glab config set editor vim
			- glab config edit

			Open the local configuration file with the default editor
			- glab config edit -l
		`),
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			var configPath string

			if isLocal {
				configPath = config.LocalConfigFile()
			} else {
				configPath = fmt.Sprintf("%s/config.yml", config.ConfigDir())
			}

			editor, err := cmdutils.GetEditor(f.Config)
			if err != nil {
				return err
			}

			editorCommand, err := browser.Command(configPath, editor)
			if err != nil {
				return err
			}

			editorCommand.Stdin = cmd.InOrStdin()
			editorCommand.Stdout = cmd.OutOrStdout()
			editorCommand.Stderr = cmd.ErrOrStderr()

			err = editorCommand.Run()
			if err != nil {
				return err
			}

			return nil
		},
	}

	cmd.Flags().BoolVarP(&isLocal, "local", "l", false, "Open '.git/glab-cli/config.yml' file instead of the global '~/.config/glab-cli/config.yml' file.")
	return cmd
}
