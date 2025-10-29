package list

import (
	"strings"

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

	showKeyIDs bool
}

func NewCmdList(f cmdutils.Factory) *cobra.Command {
	opts := &options{
		io:           f.IO(),
		gitlabClient: f.GitLabClient,
	}
	cmd := &cobra.Command{
		Use:   "list",
		Short: "Get a list of GPG keys for the currently authenticated user.",
		Long:  "",
		Example: heredoc.Doc(`
			$ glab gpg-key list
		`),
		Args: cobra.MaximumNArgs(1),
		Annotations: map[string]string{
			mcpannotations.Safe: "true",
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			return opts.run()
		},
	}

	cmd.Flags().BoolVarP(&opts.showKeyIDs, "show-id", "", false, "Shows IDs of GPG keys.")

	return cmd
}

func (o *options) run() error {
	client, err := o.gitlabClient()
	if err != nil {
		return err
	}

	keys, _, err := client.Users.ListGPGKeys()
	if err != nil {
		return cmdutils.WrapError(err, "failed to list GPG keys.")
	}

	cs := o.io.Color()
	table := tableprinter.NewTablePrinter()
	isTTy := o.io.IsOutputTTY()

	if len(keys) > 0 {
		if o.showKeyIDs {
			table.AddRow("ID", "Key", "Created At")
		} else {
			table.AddRow("Key", "Created At")
		}
	}

	for _, key := range keys {
		createdAt := key.CreatedAt.String()
		if isTTy {
			createdAt = utils.TimeToPrettyTimeAgo(*key.CreatedAt)
		}
		// Replace newlines in key with spaces for better table display
		keyDisplay := strings.ReplaceAll(key.Key, "\n", " ")
		if o.showKeyIDs {
			table.AddRow(key.ID, keyDisplay, cs.Gray(createdAt))
		} else {
			table.AddRow(keyDisplay, cs.Gray(createdAt))
		}
	}

	o.io.LogInfo(table.String())

	return nil
}
