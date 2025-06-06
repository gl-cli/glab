package delete

import (
	"fmt"
	"strconv"

	"github.com/MakeNowJust/heredoc/v2"
	"github.com/spf13/cobra"

	"gitlab.com/gitlab-org/cli/commands/cmdutils"
	"gitlab.com/gitlab-org/cli/pkg/iostreams"
)

type DeleteOpts struct {
	IO    *iostreams.IOStreams
	KeyID int
}

func NewCmdDelete(f cmdutils.Factory) *cobra.Command {
	opts := &DeleteOpts{
		IO: f.IO,
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
			httpClient, err := f.HttpClient()
			if err != nil {
				return err
			}

			baseRepo, err := f.BaseRepo()
			if err != nil {
				return err
			}

			if len(args) == 1 {
				strInt, err := strconv.Atoi(args[0])
				if err != nil {
					return fmt.Errorf("Deploy key ID must be an integer: %s", args[0])
				}
				opts.KeyID = strInt
			}

			_, err = httpClient.DeployKeys.DeleteDeployKey(baseRepo.FullName(), opts.KeyID)
			if err != nil {
				return cmdutils.WrapError(err, "deleting deploy key.")
			}

			if opts.IO.IsOutputTTY() {
				cs := opts.IO.Color()
				opts.IO.Logf("%s Deploy key deleted.\n", cs.GreenCheck())
			} else {
				opts.IO.Logf("Deploy key deleted.\n")
			}

			return nil
		},
	}

	return cmd
}
