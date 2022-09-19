package get

import (
	"fmt"
	"github.com/MakeNowJust/heredoc"
	"github.com/profclems/glab/api"
	"github.com/profclems/glab/commands/cmdutils"
	"github.com/profclems/glab/commands/variable/variableutils"
	"github.com/profclems/glab/internal/glrepo"
	"github.com/profclems/glab/pkg/iostreams"
	"github.com/spf13/cobra"
	"github.com/xanzy/go-gitlab"
)

type GetOps struct {
	HTTPClient func() (*gitlab.Client, error)
	IO         *iostreams.IOStreams
	BaseRepo   func() (glrepo.Interface, error)

	Key   string
	Group string
}

func NewCmdSet(f *cmdutils.Factory, runE func(opts *GetOps) error) *cobra.Command {
	opts := &GetOps{
		IO: f.IO,
	}

	cmd := &cobra.Command{
		Use:   "get <key>",
		Short: "get a project or group variable",
		Args:  cobra.RangeArgs(1, 1),
		Example: heredoc.Doc(`
			$ glab variable get VAR_KEY
            $ glab variable get -g GROUP VAR_KEY
		`),
		RunE: func(cmd *cobra.Command, args []string) (err error) {

			opts.HTTPClient = f.HttpClient
			opts.BaseRepo = f.BaseRepo

			opts.Key = args[0]

			if !variableutils.IsValidKey(opts.Key) {
				err = cmdutils.FlagError{Err: fmt.Errorf("invalid key provided.\n%s", variableutils.ValidKeyMsg)}
				return
			}

			if runE != nil {
				err = runE(opts)
				return
			}
			err = getRun(opts)
			return
		},
	}

	cmd.Flags().StringVarP(&opts.Group, "group", "g", "", "Get variable for a group")
	return cmd
}

func getRun(opts *GetOps) error {
	httpClient, err := opts.HTTPClient()
	if err != nil {
		return err
	}

	var variableValue string

	if opts.Group != "" {
		variable, err := api.GetGroupVariable(httpClient, opts.Group, opts.Key, nil)
		if err != nil {
			return err
		}
		variableValue = variable.Value
	} else {
		baseRepo, err := opts.BaseRepo()
		if err != nil {
			return err
		}

		variable, err := api.GetProjectVariable(httpClient, baseRepo.FullName(), opts.Key, nil)
		if err != nil {
			return err
		}
		variableValue = variable.Value
	}

	fmt.Fprint(opts.IO.StdOut, variableValue)
	return nil
}
