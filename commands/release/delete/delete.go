package delete

import (
	"fmt"
	"net/http"

	"github.com/MakeNowJust/heredoc/v2"
	"github.com/spf13/cobra"
	"github.com/xanzy/go-gitlab"
	"gitlab.com/gitlab-org/cli/commands/cmdutils"
	"gitlab.com/gitlab-org/cli/commands/release/releaseutils/upload"
	"gitlab.com/gitlab-org/cli/internal/config"
	"gitlab.com/gitlab-org/cli/internal/glrepo"
	"gitlab.com/gitlab-org/cli/pkg/iostreams"
	"gitlab.com/gitlab-org/cli/pkg/prompt"
)

type DeleteOpts struct {
	ForceDelete bool
	DeleteTag   bool
	TagName     string

	AssetLinks []*upload.ReleaseAsset
	AssetFiles []*upload.ReleaseFile

	IO         *iostreams.IOStreams
	HTTPClient func() (*gitlab.Client, error)
	BaseRepo   func() (glrepo.Interface, error)
	Config     func() (config.Config, error)
}

func NewCmdDelete(f *cmdutils.Factory) *cobra.Command {
	opts := &DeleteOpts{
		IO:     f.IO,
		Config: f.Config,
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
			$ glab release delete v1.1.0'

			# Skip the confirmation prompt and force delete
			$ glab release delete v1.0.1 -y

			# Delete release and associated tag
			$ glab release delete v1.0.1 --with-tag
`),
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.HTTPClient = f.HttpClient
			opts.BaseRepo = f.BaseRepo

			opts.TagName = args[0]

			if !opts.ForceDelete && !opts.IO.PromptEnabled() {
				return &cmdutils.FlagError{Err: fmt.Errorf("--yes or -y flag is required when not running interactively.")}
			}

			return deleteRun(opts)
		},
	}

	cmd.Flags().BoolVarP(&opts.ForceDelete, "yes", "y", false, "Skip the confirmation prompt.")
	cmd.Flags().BoolVarP(&opts.DeleteTag, "with-tag", "t", false, "Delete the associated tag.")

	return cmd
}

func deleteRun(opts *DeleteOpts) error {
	client, err := opts.HTTPClient()
	if err != nil {
		return err
	}

	repo, err := opts.BaseRepo()
	if err != nil {
		return err
	}
	color := opts.IO.Color()
	var resp *gitlab.Response

	opts.IO.Logf("%s Validating tag %s=%s %s=%s\n",
		color.ProgressIcon(),
		color.Blue("repo"), repo.FullName(),
		color.Blue("tag"), opts.TagName)

	release, resp, err := client.Releases.GetRelease(repo.FullName(), opts.TagName)
	if err != nil {
		if resp != nil && (resp.StatusCode == http.StatusNotFound || resp.StatusCode == http.StatusForbidden) {
			return cmdutils.WrapError(err, fmt.Sprintf("no release found for %q.", repo.FullName()))
		}
		return cmdutils.WrapError(err, "failed to fetch release.")
	}

	if !opts.ForceDelete && opts.IO.PromptEnabled() {
		opts.IO.Logf("This action will permanently delete release %q immediately.\n\n", release.TagName)
		err = prompt.Confirm(&opts.ForceDelete, fmt.Sprintf("Are you ABSOLUTELY SURE you wish to delete this release %q?", release.Name), false)
		if err != nil {
			return cmdutils.WrapError(err, "could not prompt")
		}
	}

	if !opts.ForceDelete {
		return cmdutils.CancelError()
	}

	opts.IO.Logf("%s Deleting release %s=%s %s=%s\n",
		color.ProgressIcon(),
		color.Blue("repo"), repo.FullName(),
		color.Blue("tag"), opts.TagName)

	release, _, err = client.Releases.DeleteRelease(repo.FullName(), release.TagName)
	if err != nil {
		return cmdutils.WrapError(err, "failed to delete release.")
	}

	opts.IO.Logf(color.Bold("%s Release %q deleted.\n"), color.RedCheck(), release.Name)

	if opts.DeleteTag {

		opts.IO.Logf("%s Deleting associated tag %q.\n",
			color.ProgressIcon(), opts.TagName)

		_, err = client.Tags.DeleteTag(repo.FullName(), release.TagName)
		if err != nil {
			return cmdutils.WrapError(err, "failed to delete tag.")
		}

		opts.IO.Logf(color.Bold("%s Tag %q deleted.\n"), color.RedCheck(), release.Name)
	}
	return nil
}
