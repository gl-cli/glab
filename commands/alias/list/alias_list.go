package list

import (
	"fmt"
	"sort"

	"gitlab.com/gitlab-org/cli/pkg/iostreams"

	"gitlab.com/gitlab-org/cli/pkg/tableprinter"

	"gitlab.com/gitlab-org/cli/commands/cmdutils"
	"gitlab.com/gitlab-org/cli/internal/config"

	"github.com/spf13/cobra"
)

type options struct {
	config func() (config.Config, error)
	io     *iostreams.IOStreams
}

func NewCmdList(f cmdutils.Factory, runF func(*options) error) *cobra.Command {
	opts := &options{
		config: f.Config,
		io:     f.IO(),
	}

	aliasListCmd := &cobra.Command{
		Use:   "list [flags]",
		Short: `List the available aliases.`,
		Long:  ``,
		RunE: func(cmd *cobra.Command, args []string) error {
			if runF != nil {
				return runF(opts)
			}
			return opts.run()
		},
	}
	return aliasListCmd
}

func (o *options) run() error {
	cfg, err := o.config()
	if err != nil {
		return err
	}

	aliasCfg, err := cfg.Aliases()
	if err != nil {
		return fmt.Errorf("couldn't read aliases config: %w", err)
	}

	if aliasCfg.Empty() {

		fmt.Fprintf(o.io.StdErr, "no aliases configured.\n")
		return nil
	}

	table := tableprinter.NewTablePrinter()
	table.MaxColWidth = 70

	aliasMap := aliasCfg.All()
	var keys []string
	for alias := range aliasMap {
		keys = append(keys, alias)
	}
	sort.Strings(keys)

	table.AddRow("Alias", "Command")
	for _, alias := range keys {
		table.AddRow(alias, aliasMap[alias])
	}
	fmt.Fprintf(o.io.StdOut, "%s", table.Render())

	return nil
}
