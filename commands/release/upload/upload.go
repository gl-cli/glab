package upload

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/MakeNowJust/heredoc/v2"
	"github.com/spf13/cobra"
	"github.com/xanzy/go-gitlab"
	"gitlab.com/gitlab-org/cli/commands/cmdutils"
	"gitlab.com/gitlab-org/cli/commands/release/releaseutils"
	"gitlab.com/gitlab-org/cli/commands/release/releaseutils/upload"
	"gitlab.com/gitlab-org/cli/internal/config"
	"gitlab.com/gitlab-org/cli/internal/glrepo"
	"gitlab.com/gitlab-org/cli/pkg/iostreams"
)

type UploadOpts struct {
	TagName          string
	AssetLinksAsJson string

	AssetLinks []*upload.ReleaseAsset
	AssetFiles []*upload.ReleaseFile

	IO         *iostreams.IOStreams
	HTTPClient func() (*gitlab.Client, error)
	BaseRepo   func() (glrepo.Interface, error)
	Config     func() (config.Config, error)
}

func NewCmdUpload(f *cmdutils.Factory) *cobra.Command {
	opts := &UploadOpts{
		IO:     f.IO,
		Config: f.Config,
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
				if len(args) < 2 && opts.AssetLinksAsJson == "" {
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
			var err error
			opts.HTTPClient = f.HttpClient
			opts.BaseRepo = f.BaseRepo

			opts.TagName = args[0]

			opts.AssetFiles, err = releaseutils.AssetsFromArgs(args[1:])
			if err != nil {
				return err
			}

			if opts.AssetFiles == nil && opts.AssetLinksAsJson == "" {
				return cmdutils.FlagError{Err: errors.New("no files specified.")}
			}

			if opts.AssetLinksAsJson != "" {
				err := json.Unmarshal([]byte(opts.AssetLinksAsJson), &opts.AssetLinks)
				if err != nil {
					return fmt.Errorf("failed to parse JSON string: %w", err)
				}
			}

			return uploadRun(opts)
		},
	}

	cmd.Flags().StringVarP(&opts.AssetLinksAsJson, "assets-links", "a", "", "`JSON` string representation of assets links, like: `--assets-links='[{\"name\": \"Asset1\", \"url\":\"https://<domain>/some/location/1\", \"link_type\": \"other\", \"direct_asset_path\": \"path/to/file\"}]'.`")

	return cmd
}

func uploadRun(opts *UploadOpts) error {
	start := time.Now()

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
			return cmdutils.WrapError(err, "release does not exist. Create a new release with `glab release create "+opts.TagName+"`.")
		}
		return cmdutils.WrapError(err, "failed to fetch release.")
	}

	// upload files and create asset links
	err = releaseutils.CreateReleaseAssets(opts.IO, client, opts.AssetFiles, opts.AssetLinks, repo.FullName(), release.TagName)
	if err != nil {
		return cmdutils.WrapError(err, "creating release assets failed.")
	}

	opts.IO.Logf(color.Bold("%s Upload succeeded after %0.2fs.\n"), color.GreenCheck(), time.Since(start).Seconds())
	return nil
}
