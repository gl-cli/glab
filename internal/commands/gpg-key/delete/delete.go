package delete

import (
	"fmt"

	"github.com/MakeNowJust/heredoc/v2"
	"github.com/spf13/cobra"
	gitlab "gitlab.com/gitlab-org/api/client-go"
	"gitlab.com/gitlab-org/cli/internal/cmdutils"
	"gitlab.com/gitlab-org/cli/internal/iostreams"
	"gitlab.com/gitlab-org/cli/internal/mcpannotations"
	"gitlab.com/gitlab-org/cli/internal/utils"
)

type options struct {
	gitlabClient func() (*gitlab.Client, error)
	io           *iostreams.IOStreams

	keyID int
}

func NewCmdDelete(f cmdutils.Factory) *cobra.Command {
	opts := &options{
		io:           f.IO(),
		gitlabClient: f.GitLabClient,
	}
	cmd := &cobra.Command{
		Use:   "delete <key-id>",
		Short: "Deletes a single GPG key specified by the ID.",
		Long:  ``,
		Example: heredoc.Doc(`
			# Delete GPG key with ID as argument
			$ glab gpg-key delete 7750633`),
		Args: cobra.ExactArgs(1),
		Annotations: map[string]string{
			mcpannotations.Destructive: "true",
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
		o.keyID = utils.StringToInt(args[0])
	}

	return nil
}

func (o *options) run() error {
	client, err := o.gitlabClient()
	if err != nil {
		return err
	}

	_, err = client.Users.DeleteGPGKey(o.keyID)
	if err != nil {
		return cmdutils.WrapError(err, "failed to delete GPG key.")
	}

	cs := o.io.Color()
	fmt.Fprintf(o.io.StdOut, "%s GPG key deleted.", cs.GreenCheck())

	return nil
}
