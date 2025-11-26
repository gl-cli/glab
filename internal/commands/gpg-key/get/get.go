package get

import (
	"github.com/MakeNowJust/heredoc/v2"
	"github.com/spf13/cobra"

	gitlab "gitlab.com/gitlab-org/api/client-go"

	"gitlab.com/gitlab-org/cli/internal/cmdutils"
	"gitlab.com/gitlab-org/cli/internal/iostreams"
	"gitlab.com/gitlab-org/cli/internal/mcpannotations"
	"gitlab.com/gitlab-org/cli/internal/tableprinter"
	"gitlab.com/gitlab-org/cli/internal/utils"
)

type options struct {
	gitlabClient func() (*gitlab.Client, error)
	io           *iostreams.IOStreams

	keyID int64
}

func NewCmdGet(f cmdutils.Factory) *cobra.Command {
	opts := &options{
		io:           f.IO(),
		gitlabClient: f.GitLabClient,
	}
	cmd := &cobra.Command{
		Use:   "get <key-id>",
		Short: "Returns a single GPG key specified by the ID.",
		Long:  ``,
		Example: heredoc.Doc(`
			# Get GPG key with ID as argument
			$ glab gpg-key get 7750633`),
		Args: cobra.ExactArgs(1),
		Annotations: map[string]string{
			mcpannotations.Safe: "true",
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := opts.complete(args); err != nil {
				return err
			}

			return opts.run()
		},
	}

	return cmd
}

func (o *options) complete(args []string) error {
	if len(args) == 1 {
		o.keyID = int64(utils.StringToInt(args[0]))
	}

	return nil
}

func (o *options) run() error {
	client, err := o.gitlabClient()
	if err != nil {
		return err
	}

	key, _, err := client.Users.GetGPGKey(o.keyID)
	if err != nil {
		return cmdutils.WrapError(err, "failed to get GPG key.")
	}

	o.io.LogInfof("Showing GPG key with ID %d\n", key.ID)

	if key.ID != 0 {
		table := tableprinter.NewTablePrinter()
		table.AddRow("ID", key.ID)
		table.AddRow("Key", key.Key)
		table.AddRow("Created At", utils.TimeToPrettyTimeAgo(*key.CreatedAt))
		o.io.LogInfo(table.String())
	} else {
		o.io.LogInfo("GPG key does not exist.")
	}

	return nil
}
