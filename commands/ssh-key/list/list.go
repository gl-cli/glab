package list

import (
	"github.com/MakeNowJust/heredoc"
	"github.com/spf13/cobra"
	"github.com/xanzy/go-gitlab"
	"gitlab.com/gitlab-org/cli/commands/cmdutils"
	"gitlab.com/gitlab-org/cli/internal/glrepo"
	"gitlab.com/gitlab-org/cli/pkg/iostreams"
	"gitlab.com/gitlab-org/cli/pkg/tableprinter"
	"gitlab.com/gitlab-org/cli/pkg/utils"
)

type ListOpts struct {
	HTTPClient func() (*gitlab.Client, error)
	IO         *iostreams.IOStreams
	BaseRepo   func() (glrepo.Interface, error)

	ShowKeyIDs bool
}

func NewCmdList(f *cmdutils.Factory, runE func(*ListOpts) error) *cobra.Command {
	opts := &ListOpts{
		IO: f.IO,
	}
	cmd := &cobra.Command{
		Use:   "list",
		Short: "Lists currently authenticated user’s SSH keys",
		Long:  "Get a list of currently authenticated user’s SSH keys",
		Example: heredoc.Doc(`
		$ glab ssh-key list
		`),
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.HTTPClient = f.HttpClient
			opts.BaseRepo = f.BaseRepo

			if runE != nil {
				return runE(opts)
			}

			return listRun(opts)
		},
	}

	cmd.Flags().BoolVarP(&opts.ShowKeyIDs, "show-id", "", false, "Show IDs of SSH Keys")

	return cmd
}

func listRun(opts *ListOpts) error {
	httpClient, err := opts.HTTPClient()
	if err != nil {
		return err
	}

	keys, _, err := httpClient.Users.ListSSHKeys()
	if err != nil {
		return cmdutils.WrapError(err, "failed to get ssh keys")
	}

	cs := opts.IO.Color()
	table := tableprinter.NewTablePrinter()
	isTTy := opts.IO.IsOutputTTY()

	for _, key := range keys {
		createdAt := key.CreatedAt.String()
		if opts.ShowKeyIDs {
			table.AddCell(key.ID)
		}
		if isTTy {
			createdAt = utils.TimeToPrettyTimeAgo(*key.CreatedAt)
		}
		table.AddRow(key.Title, key.Key, cs.Gray(createdAt))
	}

	opts.IO.LogInfo(table.String())

	return nil
}
