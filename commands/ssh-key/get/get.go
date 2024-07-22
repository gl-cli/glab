package get

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

type GetOpts struct {
	HTTPClient func() (*gitlab.Client, error)
	IO         *iostreams.IOStreams
	BaseRepo   func() (glrepo.Interface, error)

	KeyID   int
	PerPage int
	Page    int
}

func NewCmdGet(f *cmdutils.Factory, runE func(*GetOpts) error) *cobra.Command {
	opts := &GetOpts{
		IO: f.IO,
	}
	cmd := &cobra.Command{
		Use:   "get <key-id>",
		Short: "Returns a single SSH key specified by the ID.",
		Long:  ``,
		Example: heredoc.Doc(
			`
		# Get ssh key with ID as argument
		$ glab ssh-key get 7750633

		# Interactive
		$ glab ssh-key get

		# Interactive, with pagination
		$ glab ssh-key get -P 50 -p 2
		`,
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

			return getRun(opts)
		},
	}

	cmd.Flags().IntVarP(&opts.Page, "page", "p", 1, "Page number.")
	cmd.Flags().IntVarP(&opts.PerPage, "per-page", "P", 20, "Number of items to list per page.")

	return cmd
}

func getRun(opts *GetOpts) error {
	httpClient, err := opts.HTTPClient()
	if err != nil {
		return err
	}

	key, _, err := httpClient.Users.GetSSHKey(opts.KeyID)
	if err != nil {
		return cmdutils.WrapError(err, "getting SSH key.")
	}

	opts.IO.LogInfo(key.Key)

	return nil
}

func keySelectPrompt(opts *GetOpts) (int, error) {
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

	keys, response, err := httpClient.Users.ListSSHKeys(sshKeyListOptions)
	if err != nil {
		return 0, cmdutils.WrapError(err, "Retrieving list of SSH keys to prompt with.")
	}
	if len(keys) == 0 {
		return 0, cmdutils.WrapError(errors.New("no keys were found"), "Retrieving list of SSH keys.")
	}

	keyOpts := map[string]int{}
	surveyOpts := make([]string, 0, len(keys))
	for _, key := range keys {
		keyOpts[key.Title] = key.ID
		surveyOpts = append(surveyOpts, key.Title)
	}

	keySelectQuestion := &survey.Select{
		Message: fmt.Sprintf(
			"Select key - Showing %d/%d keys - page %d/%d",
			len(keys),
			response.TotalItems,
			opts.Page,
			response.TotalPages,
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
