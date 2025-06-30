package upload

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/MakeNowJust/heredoc/v2"
	"github.com/spf13/cobra"
	gitlab "gitlab.com/gitlab-org/api/client-go"
	"gitlab.com/gitlab-org/cli/internal/cmdutils"
	"gitlab.com/gitlab-org/cli/internal/commands/release/releaseutils"
	"gitlab.com/gitlab-org/cli/internal/commands/release/releaseutils/upload"
	"gitlab.com/gitlab-org/cli/internal/glrepo"
	"gitlab.com/gitlab-org/cli/internal/iostreams"
)

type options struct {
	tagName          string
	assetLinksAsJSON string

	assetLinks []*upload.ReleaseAsset
	assetFiles []*upload.ReleaseFile

	io         *iostreams.IOStreams
	httpClient func() (*gitlab.Client, error)
	baseRepo   func() (glrepo.Interface, error)
}

func NewCmdUpload(f cmdutils.Factory) *cobra.Command {
	opts := &options{
		io:         f.IO(),
		httpClient: f.HttpClient,
		baseRepo:   f.BaseRepo,
	}

	cmd := &cobra.Command{
		Use:   "upload <tag> [<files>...]",
		Short: "Upload release asset files or links to a GitLab release.",
		Long: heredoc.Doc(`Upload release assets to a GitLab release.

		Define the display name by appending '#' after the filename.
		The link type comes after the display name, like this: 'myfile.tar.gz#My display name#package'
		`),
		Args: func() cobra.PositionalArgs {
			return func(cmd *cobra.Command, args []string) error {
				if len(args) < 1 {
					return &cmdutils.FlagError{Err: errors.New("no tag name provided.")}
				}
				if len(args) < 2 && opts.assetLinksAsJSON == "" {
					return cmdutils.FlagError{Err: errors.New("no files specified.")}
				}
				return nil
			}
		}(),
		Example: heredoc.Doc(`
			# Upload a release asset with a display name. 'Type' defaults to 'other'.
			$ glab release upload v1.0.1 '/path/to/asset.zip#My display label'

			# Upload a release asset with a display name and type.
			$ glab release upload v1.0.1 '/path/to/asset.png#My display label#image'

			# Upload all assets in a specified folder. 'Type' defaults to 'other'.
			$ glab release upload v1.0.1 ./dist/*

			# Upload all tarballs in a specified folder. 'Type' defaults to 'other'.
			$ glab release upload v1.0.1 ./dist/*.tar.gz

			# Upload release assets links specified as JSON string
			$ glab release upload v1.0.1 --assets-links='
			  [
			    {
			      "name": "Asset1",
			      "url":"https://<domain>/some/location/1",
			      "link_type": "other",
			      "direct_asset_path": "path/to/file"
			    }
			  ]'
		`),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := opts.complete(args); err != nil {
				return err
			}

			if err := opts.validate(); err != nil {
				return err
			}

			return opts.run()
		},
	}

	cmd.Flags().StringVarP(&opts.assetLinksAsJSON, "assets-links", "a", "", "`JSON` string representation of assets links, like: `--assets-links='[{\"name\": \"Asset1\", \"url\":\"https://<domain>/some/location/1\", \"link_type\": \"other\", \"direct_asset_path\": \"path/to/file\"}]'.`")

	return cmd
}

func (o *options) complete(args []string) error {
	o.tagName = args[0]

	assetFiles, err := releaseutils.AssetsFromArgs(args[1:])
	if err != nil {
		return err
	}
	o.assetFiles = assetFiles

	return nil
}

func (o *options) validate() error {
	if o.assetFiles == nil && o.assetLinksAsJSON == "" {
		return cmdutils.FlagError{Err: errors.New("no files specified.")}
	}

	if o.assetLinksAsJSON != "" {
		err := json.Unmarshal([]byte(o.assetLinksAsJSON), &o.assetLinks)
		if err != nil {
			return fmt.Errorf("failed to parse JSON string: %w", err)
		}
	}

	return nil
}

func (o *options) run() error {
	start := time.Now()

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
			return cmdutils.WrapError(err, "release does not exist. Create a new release with `glab release create "+o.tagName+"`.")
		}
		return cmdutils.WrapError(err, "failed to fetch release.")
	}

	// upload files and create asset links
	err = releaseutils.CreateReleaseAssets(o.io, client, o.assetFiles, o.assetLinks, repo.FullName(), release.TagName)
	if err != nil {
		return cmdutils.WrapError(err, "creating release assets failed.")
	}

	o.io.Logf(color.Bold("%s Upload succeeded after %0.2fs.\n"), color.GreenCheck(), time.Since(start).Seconds())
	return nil
}
