package delete

import (
	"errors"
	"fmt"

	"github.com/AlecAivazis/survey/v2"
	"github.com/MakeNowJust/heredoc/v2"
	"github.com/spf13/cobra"
	gitlab "gitlab.com/gitlab-org/api/client-go"

	"gitlab.com/gitlab-org/cli/internal/api"
	"gitlab.com/gitlab-org/cli/internal/cmdutils"
	"gitlab.com/gitlab-org/cli/internal/config"
	"gitlab.com/gitlab-org/cli/internal/iostreams"
	"gitlab.com/gitlab-org/cli/internal/prompt"
	"gitlab.com/gitlab-org/cli/internal/utils"
)

type options struct {
	apiClient func(repoHost string, cfg config.Config) (*api.Client, error)
	config    config.Config
	io        *iostreams.IOStreams

	keyID   int
	perPage int
	page    int
}

func NewCmdDelete(f cmdutils.Factory) *cobra.Command {
	opts := &options{
		io:        f.IO(),
		apiClient: f.ApiClient,
		config:    f.Config(),
	}
	cmd := &cobra.Command{
		Use:   "delete <key-id>",
		Short: "Deletes a single SSH key specified by the ID.",
		Long:  ``,
		Example: heredoc.Doc(`
			# Delete SSH key with ID as argument
			$ glab ssh-key delete 7750633

			# Interactive
			$ glab ssh-key delete

			# Interactive, with pagination
			$ glab ssh-key delete -P 50 -p 2`,
		),
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := opts.complete(args); err != nil {
				return err
			}
			return opts.run()
		},
	}

	cmd.Flags().IntVarP(&opts.page, "page", "p", 1, "Page number.")
	cmd.Flags().IntVarP(&opts.perPage, "per-page", "P", 30, "Number of items to list per page.")

	return cmd
}

func (o *options) complete(args []string) error {
	if len(args) == 0 {
		keyID, err := keySelectPrompt(o)
		if err != nil {
			return err
		}
		o.keyID = keyID
	}

	if len(args) == 1 {
		o.keyID = utils.StringToInt(args[0])
	}

	return nil
}

func (o *options) run() error {
	c, err := o.apiClient("", o.config)
	if err != nil {
		return err
	}
	client := c.Lab()

	_, err = client.Users.DeleteSSHKey(o.keyID)
	if err != nil {
		return cmdutils.WrapError(err, "deleting SSH key.")
	}

	if o.io.IsOutputTTY() {
		cs := o.io.Color()
		o.io.Logf("%s SSH key deleted.\n", cs.GreenCheck())
	} else {
		o.io.Logf("SSH key deleted.\n")
	}

	return nil
}

func keySelectPrompt(opts *options) (int, error) {
	if !opts.io.PromptEnabled() {
		return 0, cmdutils.FlagError{Err: errors.New("the <key-id> argument is required when prompts are disabled.")}
	}

	sshKeyListOptions := &gitlab.ListSSHKeysOptions{
		PerPage: opts.perPage,
		Page:    opts.page,
	}

	c, err := opts.apiClient("", opts.config)
	if err != nil {
		return 0, err
	}
	client := c.Lab()

	keys, resp, err := client.Users.ListSSHKeys(sshKeyListOptions)
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
			opts.page,
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
