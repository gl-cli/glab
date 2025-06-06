package logout

import (
	"errors"
	"fmt"

	"gitlab.com/gitlab-org/cli/pkg/iostreams"

	"github.com/MakeNowJust/heredoc/v2"
	"github.com/spf13/cobra"

	"gitlab.com/gitlab-org/cli/commands/cmdutils"
	"gitlab.com/gitlab-org/cli/internal/config"
)

type LogoutOptions struct {
	IO       *iostreams.IOStreams
	Config   func() (config.Config, error)
	Hostname string
}

var opts *LogoutOptions

func NewCmdLogout(f cmdutils.Factory) *cobra.Command {
	opts = &LogoutOptions{
		IO:       f.IO,
		Config:   f.Config,
		Hostname: "",
	}

	cmd := &cobra.Command{
		Use:   "logout",
		Args:  cobra.ExactArgs(0),
		Short: "Logout from a GitLab instance.",
		Long: heredoc.Docf(`
			Logout from a GitLab instance.
			Configuration and credentials are stored in the global configuration file (default %[1]s~/.config/glab-cli/config.yml%[1]s)
		`, "`"),
		Example: heredoc.Doc(`
			Logout of a specific instance
			- glab auth logout --hostname gitlab.example.com
		`),
		RunE: func(cmd *cobra.Command, args []string) error {
			if opts.Hostname == "" {
				return &cmdutils.FlagError{Err: errors.New("hostname is required to logout. Use --hostname flag to specify hostname")}
			}
			cfg, err := opts.Config()
			if err != nil {
				return err
			}

			if err := cfg.Set(opts.Hostname, "token", ""); err != nil {
				return err
			}

			if err := cfg.Write(); err != nil {
				return err
			}

			fmt.Fprintf(f.IO.StdOut, "Successfully logged out of %s\n", opts.Hostname)
			return nil
		},
	}

	cmd.Flags().StringVarP(&opts.Hostname, "hostname", "h", "", "The hostname of the GitLab instance.")
	return cmd
}
