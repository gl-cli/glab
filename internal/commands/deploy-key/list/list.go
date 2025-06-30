package list

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
		Short: "Get a list of deploy keys for the current project.",
		Long:  "",
		Example: heredoc.Doc(`
		  - glab deploy-key list
		`),
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return opts.run()
		},
	}

	cmd.Flags().BoolVarP(&opts.showKeyIDs, "show-id", "", false, "Shows IDs of deploy keys.")
	cmd.Flags().IntVarP(&opts.page, "page", "p", 1, "Page number.")
	cmd.Flags().IntVarP(&opts.perPage, "per-page", "P", 30, "Number of items to list per page.")

	return cmd
}

func (o *options) run() error {
	httpClient, err := o.httpClient()
	if err != nil {
		return err
	}

	listProjectDeployKeysOptions := &gitlab.ListProjectDeployKeysOptions{
		Page:    o.page,
		PerPage: o.perPage,
	}

	baseRepo, err := o.baseRepo()
	if err != nil {
		return err
	}

	keys, _, err := httpClient.DeployKeys.ListProjectDeployKeys(baseRepo.FullName(), listProjectDeployKeysOptions)
	if err != nil {
		return cmdutils.WrapError(err, "failed to get deploy keys.")
	}

	cs := o.io.Color()
	table := tableprinter.NewTablePrinter()
	isTTy := o.io.IsOutputTTY()

	if len(keys) > 0 {
		if o.showKeyIDs {
			table.AddRow("ID", "Title", "Key", "Can Push", "Created At")
		} else {
			table.AddRow("Title", "Key", "Can Push", "Created At")
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
		table.AddRow(key.Title, key.Key, key.CanPush, cs.Gray(createdAt))
	}

	o.io.LogInfo(table.String())

	return nil
}
