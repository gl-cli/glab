package delete

import (
	"fmt"
	"strconv"

	"github.com/MakeNowJust/heredoc/v2"
	"github.com/spf13/cobra"

	gitlab "gitlab.com/gitlab-org/api/client-go"
	"gitlab.com/gitlab-org/cli/internal/cmdutils"
	"gitlab.com/gitlab-org/cli/internal/glrepo"
	"gitlab.com/gitlab-org/cli/internal/iostreams"
)

type options struct {
	io         *iostreams.IOStreams
	httpClient func() (*gitlab.Client, error)
	baseRepo   func() (glrepo.Interface, error)

	keyID int
}

func NewCmdDelete(f cmdutils.Factory) *cobra.Command {
	opts := &options{
		io:         f.IO(),
		httpClient: f.HttpClient,
		baseRepo:   f.BaseRepo,
	}
	cmd := &cobra.Command{
		Use:   "delete <key-id>",
		Short: "Deletes a single deploy key specified by the ID.",
		Long:  ``,
		Example: heredoc.Doc(`
			# Delete SSH key with ID as argument
			$ glab deploy-key delete 1234`,
		),
		Args: cobra.MaximumNArgs(1),
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
		strInt, err := strconv.Atoi(args[0])
		if err != nil {
			return fmt.Errorf("Deploy key ID must be an integer: %s", args[0])
		}
		o.keyID = strInt
	}

	return nil
}

func (o *options) run() error {
	httpClient, err := o.httpClient()
	if err != nil {
		return err
	}

	baseRepo, err := o.baseRepo()
	if err != nil {
		return err
	}

	_, err = httpClient.DeployKeys.DeleteDeployKey(baseRepo.FullName(), o.keyID)
	if err != nil {
		return cmdutils.WrapError(err, "deleting deploy key.")
	}

	if o.io.IsOutputTTY() {
		cs := o.io.Color()
		o.io.Logf("%s Deploy key deleted.\n", cs.GreenCheck())
	} else {
		o.io.Logf("Deploy key deleted.\n")
	}

	return nil
}
