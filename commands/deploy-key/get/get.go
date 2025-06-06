package get

import (
	"github.com/MakeNowJust/heredoc/v2"
	"github.com/spf13/cobra"
	gitlab "gitlab.com/gitlab-org/api/client-go"

	"gitlab.com/gitlab-org/cli/commands/cmdutils"
	"gitlab.com/gitlab-org/cli/internal/glrepo"
	"gitlab.com/gitlab-org/cli/pkg/iostreams"
	"gitlab.com/gitlab-org/cli/pkg/tableprinter"
	"gitlab.com/gitlab-org/cli/pkg/utils"
)

type GetOpts struct {
	HTTPClient func() (*gitlab.Client, error)
	IO         *iostreams.IOStreams
	BaseRepo   func() (glrepo.Interface, error)

	KeyID int
}

func NewCmdGet(f cmdutils.Factory) *cobra.Command {
	opts := &GetOpts{
		IO: f.IO,
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
			opts.HTTPClient = f.HttpClient
			opts.BaseRepo = f.BaseRepo
			httpClient, err := opts.HTTPClient()
			if err != nil {
				return err
			}

			baseRepo, err := opts.BaseRepo()
			if err != nil {
				return err
			}

			if len(args) == 1 {
				opts.KeyID = utils.StringToInt(args[0])
			}

			key, _, err := httpClient.DeployKeys.GetDeployKey(baseRepo.FullName(), opts.KeyID, nil)
			if err != nil {
				return cmdutils.WrapError(err, "getting deploy key.")
			}

			if key.ID != 0 {
				table := tableprinter.NewTablePrinter()
				table.AddRow("Title", "Key", "Can Push", "Created At")
				table.AddRow(key.Title, key.Key, key.CanPush, key.CreatedAt)
				opts.IO.LogInfo(table.String())
			} else {
				opts.IO.LogInfo("Deploy key does not exist.")
			}

			return nil
		},
	}

	return cmd
}
