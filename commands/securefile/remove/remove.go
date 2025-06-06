package remove

import (
	"fmt"
	"strconv"

	"github.com/MakeNowJust/heredoc/v2"
	"github.com/spf13/cobra"
	gitlab "gitlab.com/gitlab-org/api/client-go"
	"gitlab.com/gitlab-org/cli/api"
	"gitlab.com/gitlab-org/cli/commands/cmdutils"
	"gitlab.com/gitlab-org/cli/internal/config"
	"gitlab.com/gitlab-org/cli/internal/glrepo"
	"gitlab.com/gitlab-org/cli/pkg/iostreams"
	"gitlab.com/gitlab-org/cli/pkg/prompt"
)

type DeleteOpts struct {
	ForceDelete bool
	FileID      int

	IO         *iostreams.IOStreams
	HTTPClient func() (*gitlab.Client, error)
	BaseRepo   func() (glrepo.Interface, error)
	Config     func() (config.Config, error)
}

func NewCmdRemove(f cmdutils.Factory) *cobra.Command {
	opts := &DeleteOpts{
		IO:     f.IO(),
		Config: f.Config,
	}
	securefileRemoveCmd := &cobra.Command{
		Use:     "remove <fileID>",
		Short:   `Remove a secure file.`,
		Long:    ``,
		Aliases: []string{"rm", "delete"},
		Args:    cobra.ExactArgs(1),
		Example: heredoc.Doc(`
			Remove a project's secure file using the file's ID.
			- glab securefile remove 1

			Skip the confirmation prompt and force delete.
			- glab securefile remove 1 -y

			Remove a project's secure file with 'rm' alias.
			- glab securefile rm 1

			Remove a project's secure file with 'delete' alias.
			- glab securefile delete 1
		`),
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.HTTPClient = f.HttpClient
			opts.BaseRepo = f.BaseRepo

			fileID, err := strconv.Atoi(args[0])
			if err != nil {
				return fmt.Errorf("Secure file ID must be an integer: %s", args[0])
			}

			opts.FileID = fileID

			if !opts.ForceDelete && !opts.IO.PromptEnabled() {
				return &cmdutils.FlagError{Err: fmt.Errorf("--yes or -y flag is required when not running interactively.")}
			}

			return deleteRun(opts)
		},
	}

	securefileRemoveCmd.Flags().BoolVarP(&opts.ForceDelete, "yes", "y", false, "Skip the confirmation prompt.")

	return securefileRemoveCmd
}

func deleteRun(opts *DeleteOpts) error {
	apiClient, err := opts.HTTPClient()
	if err != nil {
		return err
	}

	repo, err := opts.BaseRepo()
	if err != nil {
		return err
	}

	if !opts.ForceDelete && opts.IO.PromptEnabled() {
		opts.IO.Logf("This action will permanently delete secure file %d immediately.\n\n", opts.FileID)
		err = prompt.Confirm(&opts.ForceDelete, fmt.Sprintf("Are you ABSOLUTELY SURE you wish to delete this secure file %d?", opts.FileID), false)
		if err != nil {
			return cmdutils.WrapError(err, "could not prompt")
		}
	}

	if !opts.ForceDelete {
		return cmdutils.CancelError()
	}

	color := opts.IO.Color()
	opts.IO.Logf("%s Deleting secure file %s=%s %s=%d\n",
		color.ProgressIcon(),
		color.Blue("repo"), repo.FullName(),
		color.Blue("fileID"), opts.FileID)

	err = api.RemoveSecureFile(apiClient, repo.FullName(), opts.FileID)
	if err != nil {
		return fmt.Errorf("Error removing secure file: %v", err)
	}

	opts.IO.Logf(color.Bold("%s Secure file %d deleted.\n"), color.RedCheck(), opts.FileID)

	return nil
}
