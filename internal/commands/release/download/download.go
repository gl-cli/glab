package download

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/MakeNowJust/heredoc/v2"
	"github.com/spf13/cobra"
	gitlab "gitlab.com/gitlab-org/api/client-go"
	"gitlab.com/gitlab-org/cli/internal/api"
	"gitlab.com/gitlab-org/cli/internal/cmdutils"
	"gitlab.com/gitlab-org/cli/internal/commands/release/releaseutils/upload"
	"gitlab.com/gitlab-org/cli/internal/config"
	"gitlab.com/gitlab-org/cli/internal/glrepo"
	"gitlab.com/gitlab-org/cli/internal/iostreams"
)

type options struct {
	tagName    string
	assetNames []string
	dir        string

	io         *iostreams.IOStreams
	apiClient  func(repoHost string, cfg config.Config) (*api.Client, error)
	httpClient func() (*gitlab.Client, error)
	baseRepo   func() (glrepo.Interface, error)
	config     func() config.Config
}

func NewCmdDownload(f cmdutils.Factory) *cobra.Command {
	opts := &options{
		io:         f.IO(),
		apiClient:  f.ApiClient,
		httpClient: f.HttpClient,
		baseRepo:   f.BaseRepo,
		config:     f.Config,
	}

	cmd := &cobra.Command{
		Use:   "download <tag>",
		Short: "Download asset files from a GitLab release.",
		Long: heredoc.Docf(`Download asset files from a GitLab release.

			If no tag is specified, downloads assets from the latest release.
			To specify a file name to download from the release assets, use %[1]s--asset-name%[1]s.
			%[1]s--asset-name%[1]s flag accepts glob patterns.
		`, "`"),
		Args: cobra.MaximumNArgs(1),
		Example: heredoc.Doc(`
			# Download all assets from the latest release
			$ glab release download

			# Download all assets from the specified release tag
			$ glab release download v1.1.0

			# Download assets with names matching the glob pattern
			$ glab release download v1.10.1 --asset-name="*.tar.gz"
		`),
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.complete(args)

			return opts.run(cmd.Context())
		},
	}

	cmd.Flags().StringArrayVarP(&opts.assetNames, "asset-name", "n", []string{}, "Download only assets that match the name or a glob pattern.")
	cmd.Flags().StringVarP(&opts.dir, "dir", "D", ".", "Directory to download the release assets to.")

	return cmd
}

func (o *options) complete(args []string) {
	if len(args) == 1 {
		o.tagName = args[0]
	}
}

func (o *options) run(ctx context.Context) error {
	client, err := o.httpClient()
	if err != nil {
		return err
	}
	cfg := o.config()
	repo, err := o.baseRepo()
	if err != nil {
		return err
	}
	color := o.io.Color()
	var resp *gitlab.Response
	var release *gitlab.Release
	var downloadableAssets []*upload.ReleaseAsset

	if o.tagName == "" {
		o.io.Logf("%s fetching latest release %s=%s\n",
			color.ProgressIcon(),
			color.Blue("repo"), repo.FullName())
		releases, _, err := client.Releases.ListReleases(repo.FullName(), &gitlab.ListReleasesOptions{})
		if err != nil {
			return cmdutils.WrapError(err, "could not fetch latest release.")
		}
		if len(releases) < 1 {
			return cmdutils.WrapError(errors.New("not found"), fmt.Sprintf("no release found for %q", repo.FullName()))
		}

		release = releases[0]
		o.tagName = release.TagName
	} else {
		o.io.Logf("%s fetching release %s=%s %s=%s.\n",
			color.ProgressIcon(),
			color.Blue("repo"), repo.FullName(),
			color.Blue("tag"), o.tagName)

		release, resp, err = client.Releases.GetRelease(repo.FullName(), o.tagName)
		if err != nil {
			if resp != nil && (resp.StatusCode == http.StatusNotFound || resp.StatusCode == http.StatusForbidden) {
				return cmdutils.WrapError(err, "release does not exist.")
			}
			return cmdutils.WrapError(err, "failed to fetch release.")
		}
	}

	for _, link := range release.Assets.Links {
		if len(o.assetNames) > 0 && (!matchAny(o.assetNames, link.Name)) {
			continue
		}
		downloadableAssets = append(downloadableAssets, &upload.ReleaseAsset{
			Name: &link.Name,
			URL:  &link.URL,
		})
	}

	for _, source := range release.Assets.Sources {
		source := source
		name := path.Base(source.URL)
		if len(o.assetNames) > 0 && (!matchAny(o.assetNames, name)) {
			continue
		}
		downloadableAssets = append(downloadableAssets, &upload.ReleaseAsset{
			Name: &name,
			URL:  &source.URL,
		})
	}

	if len(downloadableAssets) < 1 {
		o.io.Logf("%s no release assets found!\n",
			color.DotWarnIcon())
		return nil
	}
	o.io.Logf("%s downloading release assets %s=%s %s=%s\n",
		color.ProgressIcon(),
		color.Blue("repo"), repo.FullName(),
		color.Blue("tag"), o.tagName)

	c, err := o.apiClient(repo.RepoHost(), cfg)
	if err != nil {
		return err
	}
	err = downloadAssets(ctx, c, o.io, downloadableAssets, o.dir)
	if err != nil {
		return cmdutils.WrapError(err, "failed to download release.")
	}

	o.io.Logf(color.Bold("%s release %q downloaded\n"), color.RedCheck(), release.Name)

	return nil
}

func matchAny(patterns []string, name string) bool {
	for _, p := range patterns {
		matched, err := filepath.Match(p, name)
		if err == nil && matched {
			return true
		}
	}
	return false
}

func downloadAssets(ctx context.Context, httpClient *api.Client, io *iostreams.IOStreams, toDownload []*upload.ReleaseAsset, destDir string) error {
	color := io.Color()
	for _, asset := range toDownload {
		io.Logf("%s downloading file %s=%s %s=%s.\n",
			color.ProgressIcon(),
			color.Blue("name"), *asset.Name,
			color.Blue("url"), *asset.URL)

		var sanitizedAssetName string
		if asset.Name != nil {
			sanitizedAssetName = sanitizeAssetName(*asset.Name)
		}
		destDir, err := filepath.Abs(destDir)
		if err != nil {
			return fmt.Errorf("resolving absolute download directory path: %v", err)
		}

		destPath := filepath.Join(destDir, sanitizedAssetName)
		if !strings.HasPrefix(destPath, destDir) {
			return fmt.Errorf("invalid file path name.")
		}

		err = downloadAsset(ctx, httpClient, *asset.URL, destPath)
		if err != nil {
			return err
		}
	}

	return nil
}

func sanitizeAssetName(asset string) string {
	if !strings.HasPrefix(asset, "/") {
		// Prefix the asset with "/" ensures that filepath.Clean removes all `/..`
		// See rule 4 of filepath.Clean for more information: https://pkg.go.dev/path/filepath#Clean
		asset = "/" + asset
	}
	return filepath.Clean(asset)
}

func downloadAsset(ctx context.Context, client *api.Client, assetURL, destinationPath string) error {
	var body io.Reader
	// color := streams.Color()

	baseURL, _ := url.Parse(assetURL)

	req, err := api.NewHTTPRequest(ctx, client, http.MethodGet, baseURL, body, []string{"Accept:application/octet-stream"}, false)
	if err != nil {
		return err
	}
	resp, err := client.HTTPClient().Do(req)
	if err != nil {
		return err
	}

	defer resp.Body.Close()

	if resp.StatusCode > 299 {
		return errors.New(resp.Status)
	}

	f, err := os.OpenFile(destinationPath, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o644)
	if err != nil {
		return err
	}
	defer f.Close()

	_, err = io.Copy(f, resp.Body)
	return err
}
