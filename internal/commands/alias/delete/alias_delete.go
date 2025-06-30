package delete

import (
	"fmt"

	"gitlab.com/gitlab-org/cli/internal/iostreams"

	"gitlab.com/gitlab-org/cli/internal/cmdutils"

	"github.com/spf13/cobra"
	"gitlab.com/gitlab-org/cli/internal/config"
)

type options struct {
	config func() config.Config
	name   string
	io     *iostreams.IOStreams
}

func NewCmdDelete(f cmdutils.Factory) *cobra.Command {
	opts := &options{
		config: f.Config,
		io:     f.IO(),
	}

	aliasDeleteCmd := &cobra.Command{
		Use:   "delete <alias name> [flags]",
		Short: `Delete an alias.`,
		Long:  ``,
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.complete(args)

			return opts.run()
		},
	}
	return aliasDeleteCmd
}

func (o *options) complete(args []string) {
	o.name = args[0]
}

func (o *options) run() error {
	c := o.io.Color()
	cfg := o.config()

	aliasCfg, err := cfg.Aliases()
	if err != nil {
		return fmt.Errorf("couldn't read aliases config: %w", err)
	}

	expansion, ok := aliasCfg.Get(o.name)
	if !ok {
		return fmt.Errorf("no such alias '%s'.", o.name)
	}
	err = aliasCfg.Delete(o.name)
	if err != nil {
		return fmt.Errorf("failed to delete alias '%s': %w", o.name, err)
	}
	redCheck := c.Red("✓")
	fmt.Fprintf(o.io.StdErr, "%s Deleted alias '%s'; was '%s'.\n", redCheck, o.name, expansion)
	return nil
}
