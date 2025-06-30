package delete

import (
	"fmt"
	"net/http"

	"github.com/MakeNowJust/heredoc/v2"
	"github.com/spf13/cobra"
	gitlab "gitlab.com/gitlab-org/api/client-go"
	"gitlab.com/gitlab-org/cli/internal/cmdutils"
	"gitlab.com/gitlab-org/cli/internal/config"
	"gitlab.com/gitlab-org/cli/internal/glrepo"
	"gitlab.com/gitlab-org/cli/internal/iostreams"
	"gitlab.com/gitlab-org/cli/internal/prompt"
)

type options struct {
	forceDelete bool
	deleteTag   bool
	tagName     string

	io         *iostreams.IOStreams
	httpClient func() (*gitlab.Client, error)
	baseRepo   func() (glrepo.Interface, error)
	config     func() config.Config
}

func NewCmdDelete(f cmdutils.Factory) *cobra.Command {
	opts := &options{
		io:         f.IO(),
		httpClient: f.HttpClient,
		baseRepo:   f.BaseRepo,
		config:     f.Config,
	}

	cmd := &cobra.Command{
		Use:   "delete <tag>",
		Short: "Delete a GitLab release.",
		Long: heredoc.Docf(`Delete release assets to GitLab release. Requires the Maintainer role or higher.

			Deleting a release does not delete the associated tag, unless you specify %[1]s--with-tag%[1]s.
		`, "`"),
		Args: cmdutils.MinimumArgs(1, "no tag name provided"),
		Example: heredoc.Doc(`
			# Delete a release (with a confirmation prompt)
			$ glab release delete v1.1.0

			# Skip the confirmation prompt and force delete
			$ glab release delete v1.0.1 -y

			# Delete release and associated tag
			$ glab release delete v1.0.1 --with-tag
		`),
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.complete(args)

			if err := opts.validate(); err != nil {
				return err
			}

			return opts.run()
		},
	}

	cmd.Flags().BoolVarP(&opts.forceDelete, "yes", "y", false, "Skip the confirmation prompt.")
	cmd.Flags().BoolVarP(&opts.deleteTag, "with-tag", "t", false, "Delete the associated tag.")

	return cmd
}

func (o *options) complete(args []string) {
	o.tagName = args[0]
}

func (o *options) validate() error {
	if !o.forceDelete && !o.io.PromptEnabled() {
		return &cmdutils.FlagError{Err: fmt.Errorf("--yes or -y flag is required when not running interactively.")}
	}

	return nil
}

func (o *options) run() error {
	client, err := o.httpClient()
	if err != nil {
		return err
	}

	repo, err := o.baseRepo()
	if err != nil {
		return err
	}
	color := o.io.Color()
	var resp *gitlab.Response

	o.io.Logf("%s Validating tag %s=%s %s=%s\n",
		color.ProgressIcon(),
		color.Blue("repo"), repo.FullName(),
		color.Blue("tag"), o.tagName)

	release, resp, err := client.Releases.GetRelease(repo.FullName(), o.tagName)
	if err != nil {
		if resp != nil && (resp.StatusCode == http.StatusNotFound || resp.StatusCode == http.StatusForbidden) {
			return cmdutils.WrapError(err, fmt.Sprintf("no release found for %q.", repo.FullName()))
		}
		return cmdutils.WrapError(err, "failed to fetch release.")
	}

	if !o.forceDelete && o.io.PromptEnabled() {
		o.io.Logf("This action will permanently delete release %q immediately.\n\n", release.TagName)
		err = prompt.Confirm(&o.forceDelete, fmt.Sprintf("Are you ABSOLUTELY SURE you wish to delete this release %q?", release.Name), false)
		if err != nil {
			return cmdutils.WrapError(err, "could not prompt")
		}
	}

	if !o.forceDelete {
		return cmdutils.CancelError()
	}

	o.io.Logf("%s Deleting release %s=%s %s=%s\n",
		color.ProgressIcon(),
		color.Blue("repo"), repo.FullName(),
		color.Blue("tag"), o.tagName)

	release, _, err = client.Releases.DeleteRelease(repo.FullName(), release.TagName)
	if err != nil {
		return cmdutils.WrapError(err, "failed to delete release.")
	}

	o.io.Logf(color.Bold("%s Release %q deleted.\n"), color.RedCheck(), release.Name)

	if o.deleteTag {

		o.io.Logf("%s Deleting associated tag %q.\n",
			color.ProgressIcon(), o.tagName)

		_, err = client.Tags.DeleteTag(repo.FullName(), release.TagName)
		if err != nil {
			return cmdutils.WrapError(err, "failed to delete tag.")
		}

		o.io.Logf(color.Bold("%s Tag %q deleted.\n"), color.RedCheck(), release.Name)
	}
	return nil
}
