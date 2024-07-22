package list

import (
	"github.com/MakeNowJust/heredoc/v2"
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

	// Pagination
	Page    int
	PerPage int

	ShowKeyIDs bool
}

func NewCmdList(f *cmdutils.Factory, runE func(*ListOpts) error) *cobra.Command {
	opts := &ListOpts{
		IO: f.IO,
	}
	cmd := &cobra.Command{
		Use:   "list",
		Short: "Get a list of SSH keys for the currently authenticated user.",
		Long:  "",
		Example: heredoc.Doc(`
		glab ssh-key list
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

	cmd.Flags().BoolVarP(&opts.ShowKeyIDs, "show-id", "", false, "Shows IDs of SSH keys.")
	cmd.Flags().IntVarP(&opts.Page, "page", "p", 1, "Page number.")
	cmd.Flags().IntVarP(&opts.PerPage, "per-page", "P", 30, "Number of items to list per page.")

	return cmd
}

func listRun(opts *ListOpts) error {
	httpClient, err := opts.HTTPClient()
	if err != nil {
		return err
	}

	sshKeyListOptions := &gitlab.ListSSHKeysOptions{
		Page:    opts.Page,
		PerPage: opts.PerPage,
	}
	keys, _, err := httpClient.Users.ListSSHKeys(sshKeyListOptions)
	if err != nil {
		return cmdutils.WrapError(err, "failed to get SSH keys.")
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
