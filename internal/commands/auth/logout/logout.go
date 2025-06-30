package logout

import (
	"fmt"

	"gitlab.com/gitlab-org/cli/internal/iostreams"

	"github.com/MakeNowJust/heredoc/v2"
	"github.com/spf13/cobra"

	"gitlab.com/gitlab-org/cli/internal/cmdutils"
	"gitlab.com/gitlab-org/cli/internal/config"
)

type options struct {
	io       *iostreams.IOStreams
	config   func() config.Config
	hostname string
}

func NewCmdLogout(f cmdutils.Factory) *cobra.Command {
	opts := &options{
		io:       f.IO(),
		config:   f.Config,
		hostname: "",
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
			return opts.run()
		},
	}

	cmd.Flags().StringVarP(&opts.hostname, "hostname", "h", "", "The hostname of the GitLab instance.")
	cobra.CheckErr(cmd.MarkFlagRequired("hostname"))
	return cmd
}

func (o *options) run() error {
	cfg := o.config()

	if err := cfg.Set(o.hostname, "token", ""); err != nil {
		return err
	}

	if err := cfg.Write(); err != nil {
		return err
	}

	fmt.Fprintf(o.io.StdOut, "Successfully logged out of %s\n", o.hostname)
	return nil
}
