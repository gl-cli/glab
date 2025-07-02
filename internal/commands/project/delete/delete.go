package delete

import (
	"fmt"
	"net/http"
	"strings"

	"gitlab.com/gitlab-org/cli/internal/api"
	"gitlab.com/gitlab-org/cli/internal/iostreams"

	"github.com/MakeNowJust/heredoc/v2"
	"github.com/spf13/cobra"
	"gitlab.com/gitlab-org/cli/internal/cmdutils"
	"gitlab.com/gitlab-org/cli/internal/config"
	"gitlab.com/gitlab-org/cli/internal/glrepo"
	"gitlab.com/gitlab-org/cli/internal/prompt"
)

type options struct {
	forceDelete bool
	repoName    string
	args        []string

	io        *iostreams.IOStreams
	apiClient func(repoHost string, cfg config.Config) (*api.Client, error)
	config    config.Config
	baseRepo  func() (glrepo.Interface, error)
}

func NewCmdDelete(f cmdutils.Factory) *cobra.Command {
	opts := &options{
		io:        f.IO(),
		apiClient: f.ApiClient,
		config:    f.Config(),
		baseRepo:  f.BaseRepo,
	}

	projectCreateCmd := &cobra.Command{
		Use:   "delete [<NAMESPACE>/]<NAME>",
		Short: `Delete an existing repository on GitLab.`,
		Long:  ``,
		Args:  cobra.MaximumNArgs(1),
		Example: heredoc.Doc(`
			# Delete a personal repository.
			$ glab repo delete dotfiles

			# Delete a repository in a GitLab group, or another repository
			# you have write access to:
			$ glab repo delete mygroup/dotfiles
			$ glab repo delete myorg/mynamespace/dotfiles
	  `),
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.args = args
			return opts.run()
		},
	}

	projectCreateCmd.Flags().BoolVarP(&opts.forceDelete, "yes", "y", false, "Skip the confirmation prompt and immediately delete the repository.")

	return projectCreateCmd
}

func (o *options) run() error {
	c, err := o.apiClient("", o.config)
	if err != nil {
		return err
	}
	gitlabClient := c.Lab()

	baseRepo, baseRepoErr := o.baseRepo() // don't handle error yet
	if len(o.args) == 1 {
		o.repoName = o.args[0]

		if !strings.ContainsRune(o.repoName, '/') {
			namespace := ""
			if baseRepoErr == nil {
				namespace = baseRepo.RepoOwner()
			} else {
				currentUser, _, err := gitlabClient.Users.CurrentUser()
				if err != nil {
					return err
				}
				namespace = currentUser.Username
			}
			o.repoName = fmt.Sprintf("%s/%s", namespace, o.repoName)
		}
	} else {
		if baseRepoErr != nil {
			return baseRepoErr
		}
		o.repoName = baseRepo.FullName()
	}

	if !o.forceDelete && !o.io.PromptEnabled() {
		return &cmdutils.FlagError{Err: fmt.Errorf("--yes or -y flag is required when not running interactively.")}
	}

	if !o.forceDelete && o.io.PromptEnabled() {
		fmt.Fprintf(o.io.StdErr, "This action will permanently delete %s immediately, including its repositories and all content: issues and merge requests.\n\n", o.repoName)
		err = prompt.Confirm(&o.forceDelete, fmt.Sprintf("Are you ABSOLUTELY SURE you wish to delete %s?", o.repoName), false)
		if err != nil {
			return err
		}
	}

	if o.forceDelete {
		if o.io.IsErrTTY && o.io.IsaTTY {
			fmt.Fprintf(o.io.StdErr, "- Deleting project %s\n", o.repoName)
		}
		resp, err := gitlabClient.Projects.DeleteProject(o.repoName, nil)
		if err != nil && resp == nil {
			return err
		}
		if resp.StatusCode == http.StatusUnauthorized {
			return fmt.Errorf("you are not authorized to delete %s.\nCheck your token used for glab. Make sure it has the `api` and `write_repository` scopes enabled.", o.repoName)
		}
		return err
	}
	fmt.Fprintln(o.io.StdErr, "aborted by user")
	return nil
}
