package delete

import (
	"fmt"
	"net/http"
	"strings"

	"gitlab.com/gitlab-org/cli/pkg/iostreams"

	"github.com/MakeNowJust/heredoc/v2"
	"github.com/spf13/cobra"
	"github.com/xanzy/go-gitlab"
	"gitlab.com/gitlab-org/cli/api"
	"gitlab.com/gitlab-org/cli/commands/cmdutils"
	"gitlab.com/gitlab-org/cli/internal/glrepo"
	"gitlab.com/gitlab-org/cli/pkg/prompt"
)

type DeleteOpts struct {
	ForceDelete bool
	RepoName    string
	Args        []string

	IO       *iostreams.IOStreams
	Lab      func() (*gitlab.Client, error)
	BaseRepo func() (glrepo.Interface, error)
}

func NewCmdDelete(f *cmdutils.Factory) *cobra.Command {
	opts := &DeleteOpts{
		IO:       f.IO,
		Lab:      f.HttpClient,
		BaseRepo: f.BaseRepo,
	}

	projectCreateCmd := &cobra.Command{
		Use:   "delete [<NAMESPACE>/]<NAME>",
		Short: `Delete an existing repository on GitLab.`,
		Long:  ``,
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.Args = args
			return deleteRun(opts)
		},
		Example: heredoc.Doc(`
		# Delete a personal repository.
		$ glab repo delete dotfiles

		# Delete a repository in a GitLab group, or another repository
		# you have write access to:
		$ glab repo delete mygroup/dotfiles
		$ glab repo delete myorg/mynamespace/dotfiles
	  `),
	}

	projectCreateCmd.Flags().BoolVarP(&opts.ForceDelete, "yes", "y", false, "Skip the confirmation prompt and immediately delete the repository.")

	return projectCreateCmd
}

func deleteRun(opts *DeleteOpts) error {
	labClient, err := opts.Lab()
	if err != nil {
		return err
	}

	baseRepo, baseRepoErr := opts.BaseRepo() // don't handle error yet
	if len(opts.Args) == 1 {
		opts.RepoName = opts.Args[0]

		if !strings.ContainsRune(opts.RepoName, '/') {
			namespace := ""
			if baseRepoErr == nil {
				namespace = baseRepo.RepoOwner()
			} else {
				currentUser, err := api.CurrentUser(labClient)
				if err != nil {
					return err
				}
				namespace = currentUser.Username
			}
			opts.RepoName = fmt.Sprintf("%s/%s", namespace, opts.RepoName)
		}
	} else {
		if baseRepoErr != nil {
			return baseRepoErr
		}
		opts.RepoName = baseRepo.FullName()
	}

	if !opts.ForceDelete && !opts.IO.PromptEnabled() {
		return &cmdutils.FlagError{Err: fmt.Errorf("--yes or -y flag is required when not running interactively.")}
	}

	if !opts.ForceDelete && opts.IO.PromptEnabled() {
		fmt.Fprintf(opts.IO.StdErr, "This action will permanently delete %s immediately, including its repositories and all content: issues and merge requests.\n\n", opts.RepoName)
		err = prompt.Confirm(&opts.ForceDelete, fmt.Sprintf("Are you ABSOLUTELY SURE you wish to delete %s?", opts.RepoName), false)
		if err != nil {
			return err
		}
	}

	if opts.ForceDelete {
		if opts.IO.IsErrTTY && opts.IO.IsaTTY {
			fmt.Fprintf(opts.IO.StdErr, "- Deleting project %s\n", opts.RepoName)
		}
		resp, err := api.DeleteProject(labClient, opts.RepoName)
		if err != nil && resp == nil {
			return err
		}
		if resp.StatusCode == http.StatusUnauthorized {
			return fmt.Errorf("you are not authorized to delete %s.\nCheck your token used for glab. Make sure it has the `api` and `write_repository` scopes enabled.", opts.RepoName)
		}
		return err
	}
	fmt.Fprintln(opts.IO.StdErr, "aborted by user")
	return nil
}
