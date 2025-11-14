package logout

import (
	"fmt"

	"github.com/MakeNowJust/heredoc/v2"
	"github.com/spf13/cobra"

	"gitlab.com/gitlab-org/cli/internal/cmdutils"
	"gitlab.com/gitlab-org/cli/internal/config"
	"gitlab.com/gitlab-org/cli/internal/iostreams"
	"gitlab.com/gitlab-org/cli/internal/mcpannotations"
)

// fieldsToClear a collection of all fields to clear from the config when logging out.
var fieldsToClear = []string{
	"token",
	"job_token",
	"is_oauth2",
	"oauth2_refresh_token",
	"oauth2_expiry_date",
}

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
		Annotations: map[string]string{
			mcpannotations.Destructive: "true",
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			return opts.run()
		},
	}

	cmd.Flags().StringVarP(&opts.hostname, "hostname", "", "", "The hostname of the GitLab instance.")
	cobra.CheckErr(cmd.MarkFlagRequired("hostname"))
	return cmd
}

func (o *options) run() error {
	cfg := o.config()

	for _, key := range fieldsToClear {
		if err := cfg.Set(o.hostname, key, ""); err != nil {
			return err
		}
	}

	if err := cfg.Write(); err != nil {
		return err
	}

	fmt.Fprintf(o.io.StdOut, "Successfully logged out of %s\n", o.hostname)
	return nil
}
