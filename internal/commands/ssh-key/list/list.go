package list

import (
	"github.com/MakeNowJust/heredoc/v2"
	"github.com/spf13/cobra"
	gitlab "gitlab.com/gitlab-org/api/client-go"
	"gitlab.com/gitlab-org/cli/internal/api"
	"gitlab.com/gitlab-org/cli/internal/cmdutils"
	"gitlab.com/gitlab-org/cli/internal/config"
	"gitlab.com/gitlab-org/cli/internal/iostreams"
	"gitlab.com/gitlab-org/cli/internal/tableprinter"
	"gitlab.com/gitlab-org/cli/internal/utils"
)

type options struct {
	apiClient func(repoHost string, cfg config.Config) (*api.Client, error)
	config    config.Config
	io        *iostreams.IOStreams

	// Pagination
	page    int
	perPage int

	showKeyIDs bool
}

func NewCmdList(f cmdutils.Factory) *cobra.Command {
	opts := &options{
		io:        f.IO(),
		apiClient: f.ApiClient,
		config:    f.Config(),
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
	c, err := o.apiClient("", o.config)
	if err != nil {
		return err
	}
	client := c.Lab()

	sshKeyListOptions := &gitlab.ListSSHKeysOptions{
		Page:    o.page,
		PerPage: o.perPage,
	}
	keys, _, err := client.Users.ListSSHKeys(sshKeyListOptions)
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
