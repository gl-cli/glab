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

type ListOpts struct {
	HTTPClient func() (*gitlab.Client, error)
	IO         *iostreams.IOStreams
	BaseRepo   func() (glrepo.Interface, error)

	// Pagination
	Page    int
	PerPage int

	ShowKeyIDs bool
}

func NewCmdList(f cmdutils.Factory) *cobra.Command {
	opts := &ListOpts{
		IO: f.IO(),
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
			opts.HTTPClient = f.HttpClient
			opts.BaseRepo = f.BaseRepo

			return listRun(opts)
		},
	}

	cmd.Flags().BoolVarP(&opts.ShowKeyIDs, "show-id", "", false, "Shows IDs of deploy keys.")
	cmd.Flags().IntVarP(&opts.Page, "page", "p", 1, "Page number.")
	cmd.Flags().IntVarP(&opts.PerPage, "per-page", "P", 30, "Number of items to list per page.")

	return cmd
}

func listRun(opts *ListOpts) error {
	httpClient, err := opts.HTTPClient()
	if err != nil {
		return err
	}

	listProjectDeployKeysOptions := &gitlab.ListProjectDeployKeysOptions{
		Page:    opts.Page,
		PerPage: opts.PerPage,
	}

	baseRepo, err := opts.BaseRepo()
	if err != nil {
		return err
	}

	keys, _, err := httpClient.DeployKeys.ListProjectDeployKeys(baseRepo.FullName(), listProjectDeployKeysOptions)
	if err != nil {
		return cmdutils.WrapError(err, "failed to get deploy keys.")
	}

	cs := opts.IO.Color()
	table := tableprinter.NewTablePrinter()
	isTTy := opts.IO.IsOutputTTY()

	if len(keys) > 0 {
		if opts.ShowKeyIDs {
			table.AddRow("ID", "Title", "Key", "Can Push", "Created At")
		} else {
			table.AddRow("Title", "Key", "Can Push", "Created At")
		}
	}

	for _, key := range keys {
		createdAt := key.CreatedAt.String()
		if opts.ShowKeyIDs {
			table.AddCell(key.ID)
		}
		if isTTy {
			createdAt = utils.TimeToPrettyTimeAgo(*key.CreatedAt)
		}
		table.AddRow(key.Title, key.Key, key.CanPush, cs.Gray(createdAt))
	}

	opts.IO.LogInfo(table.String())

	return nil
}
