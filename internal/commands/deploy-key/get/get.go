package get

import (
	"github.com/MakeNowJust/heredoc/v2"
	"github.com/spf13/cobra"
	gitlab "gitlab.com/gitlab-org/api/client-go"

	"gitlab.com/gitlab-org/cli/internal/cmdutils"
	"gitlab.com/gitlab-org/cli/internal/glrepo"
	"gitlab.com/gitlab-org/cli/internal/iostreams"
	"gitlab.com/gitlab-org/cli/internal/tableprinter"
	"gitlab.com/gitlab-org/cli/internal/utils"
)

type options struct {
	httpClient func() (*gitlab.Client, error)
	io         *iostreams.IOStreams
	baseRepo   func() (glrepo.Interface, error)

	keyID int
}

func NewCmdGet(f cmdutils.Factory) *cobra.Command {
	opts := &options{
		io:         f.IO(),
		httpClient: f.HttpClient,
		baseRepo:   f.BaseRepo,
	}
	cmd := &cobra.Command{
		Use:   "get <key-id>",
		Short: "Returns a single deploy key specified by the ID.",
		Long:  ``,
		Example: heredoc.Doc(`
			# Get deploy key with ID as argument
			$ glab deploy-key get 1234
		`),
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.complete(args)

			return opts.run()
		},
	}

	return cmd
}

func (o *options) complete(args []string) {
	if len(args) == 1 {
		o.keyID = utils.StringToInt(args[0])
	}
}

func (o *options) run() error {
	httpClient, err := o.httpClient()
	if err != nil {
		return err
	}

	baseRepo, err := o.baseRepo()
	if err != nil {
		return err
	}

	key, _, err := httpClient.DeployKeys.GetDeployKey(baseRepo.FullName(), o.keyID, nil)
	if err != nil {
		return cmdutils.WrapError(err, "getting deploy key.")
	}

	if key.ID != 0 {
		table := tableprinter.NewTablePrinter()
		table.AddRow("Title", "Key", "Can Push", "Created At")
		table.AddRow(key.Title, key.Key, key.CanPush, key.CreatedAt)
		o.io.LogInfo(table.String())
	} else {
		o.io.LogInfo("Deploy key does not exist.")
	}

	return nil
}
