package list

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

type options struct {
	httpClient func() (*gitlab.Client, error)
	io         *iostreams.IOStreams
	baseRepo   func() (glrepo.Interface, error)

	// Pagination
	page    int
	perPage int

	showKeyIDs bool
}

func NewCmdList(f cmdutils.Factory) *cobra.Command {
	opts := &options{
		io:         f.IO(),
		httpClient: f.HttpClient,
		baseRepo:   f.BaseRepo,
	}
	cmd := &cobra.Command{
		Use:   "list",
		Short: "Get a list of SSH keys for the currently authenticated user.",
		Long:  "",
		Example: heredoc.Doc(`
			$ glab ssh-key list
		`),
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return opts.run()
		},
	}

	cmd.Flags().BoolVarP(&opts.showKeyIDs, "show-id", "", false, "Shows IDs of SSH keys.")
	cmd.Flags().IntVarP(&opts.page, "page", "p", 1, "Page number.")
	cmd.Flags().IntVarP(&opts.perPage, "per-page", "P", 30, "Number of items to list per page.")

	return cmd
}

func (o *options) run() error {
	httpClient, err := o.httpClient()
	if err != nil {
		return err
	}

	sshKeyListOptions := &gitlab.ListSSHKeysOptions{
		Page:    o.page,
		PerPage: o.perPage,
	}
	keys, _, err := httpClient.Users.ListSSHKeys(sshKeyListOptions)
	if err != nil {
		return cmdutils.WrapError(err, "failed to get SSH keys.")
	}

	cs := o.io.Color()
	table := tableprinter.NewTablePrinter()
	isTTy := o.io.IsOutputTTY()

	if len(keys) > 0 {
		if o.showKeyIDs {
			table.AddRow("ID", "Title", "Key", "Usage type", "Created At")
		} else {
			table.AddRow("Title", "Key", "Usage type", "Created At")
		}
	}

	for _, key := range keys {
		createdAt := key.CreatedAt.String()
		if o.showKeyIDs {
			table.AddCell(key.ID)
		}
		if isTTy {
			createdAt = utils.TimeToPrettyTimeAgo(*key.CreatedAt)
		}
		table.AddRow(key.Title, key.Key, key.UsageType, cs.Gray(createdAt))
	}

	o.io.LogInfo(table.String())

	return nil
}
