package delete

import (
	"errors"
	"fmt"

	"github.com/AlecAivazis/survey/v2"
	"github.com/MakeNowJust/heredoc/v2"
	"github.com/spf13/cobra"
	"github.com/xanzy/go-gitlab"

	"gitlab.com/gitlab-org/cli/commands/cmdutils"
	"gitlab.com/gitlab-org/cli/internal/glrepo"
	"gitlab.com/gitlab-org/cli/pkg/iostreams"
	"gitlab.com/gitlab-org/cli/pkg/prompt"
	"gitlab.com/gitlab-org/cli/pkg/utils"
)

type DeleteOpts struct {
	HTTPClient func() (*gitlab.Client, error)
	IO         *iostreams.IOStreams
	BaseRepo   func() (glrepo.Interface, error)

	KeyID   int
	PerPage int
	Page    int
}

func NewCmdDelete(f *cmdutils.Factory, runE func(*DeleteOpts) error) *cobra.Command {
	opts := &DeleteOpts{
		IO: f.IO,
	}
	cmd := &cobra.Command{
		Use:   "delete <key-id>",
		Short: "Deletes a single SSH key specified by the ID.",
		Long:  ``,
		Example: heredoc.Doc(
			`
		# Delete SSH key with ID as argument
		$ glab ssh-key delete 7750633

		# Interactive
		$ glab ssh-key delete

		# Interactive, with pagination
		$ glab ssh-key delete -P 50 -p 2`,
		),
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.HTTPClient = f.HttpClient
			opts.BaseRepo = f.BaseRepo

			if len(args) == 0 {
				keyID, err := keySelectPrompt(opts)
				if err != nil {
					return err
				}
				opts.KeyID = keyID
			}

			if len(args) == 1 {
				opts.KeyID = utils.StringToInt(args[0])
			}

			if runE != nil {
				return runE(opts)
			}

			return deleteRun(opts)
		},
	}

	cmd.Flags().IntVarP(&opts.Page, "page", "p", 1, "Page number.")
	cmd.Flags().IntVarP(&opts.PerPage, "per-page", "P", 30, "Number of items to list per page.")

	return cmd
}

func deleteRun(opts *DeleteOpts) error {
	httpClient, err := opts.HTTPClient()
	if err != nil {
		return err
	}

	_, err = httpClient.Users.DeleteSSHKey(opts.KeyID)
	if err != nil {
		return cmdutils.WrapError(err, "deleting SSH key.")
	}

	if opts.IO.IsOutputTTY() {
		cs := opts.IO.Color()
		opts.IO.Logf("%s SSH key deleted.\n", cs.GreenCheck())
	} else {
		opts.IO.Logf("SSH key deleted.\n")
	}

	return nil
}

func keySelectPrompt(opts *DeleteOpts) (int, error) {
	if !opts.IO.PromptEnabled() {
		return 0, cmdutils.FlagError{Err: errors.New("the <key-id> argument is required when prompts are disabled.")}
	}

	sshKeyListOptions := &gitlab.ListSSHKeysOptions{
		PerPage: opts.PerPage,
		Page:    opts.Page,
	}

	httpClient, err := opts.HTTPClient()
	if err != nil {
		return 0, err
	}

	keys, resp, err := httpClient.Users.ListSSHKeys(sshKeyListOptions)
	if err != nil {
		return 0, cmdutils.WrapError(err, "Retrieving list of SSH keys.")
	}
	if len(keys) == 0 {
		return 0, cmdutils.WrapError(errors.New("no keys found"), "Retrieving list of SSH keys.")
	}

	keyOpts := map[string]int{}
	surveyOpts := make([]string, 0, len(keys))
	for _, key := range keys {
		keyOpts[key.Title] = key.ID
		surveyOpts = append(surveyOpts, key.Title)
	}

	keySelectQuestion := &survey.Select{
		Message: fmt.Sprintf(
			"Select key to delete - Showing %d/%d keys - page %d/%d",
			len(keys),
			resp.TotalItems,
			opts.Page,
			resp.TotalPages,
		),
		Options: surveyOpts,
	}

	var result string
	err = prompt.AskOne(keySelectQuestion, &result)
	if err != nil {
		return 0, cmdutils.WrapError(err, "prompting for SSH key to delete.")
	}

	return keyOpts[result], nil
}
