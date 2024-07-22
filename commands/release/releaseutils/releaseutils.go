package releaseutils

import (
	"fmt"
	"io"
	"os"
	"strings"

	"gitlab.com/gitlab-org/cli/commands/cmdutils"

	"gitlab.com/gitlab-org/cli/commands/release/releaseutils/upload"
	"gitlab.com/gitlab-org/cli/internal/glrepo"

	"gitlab.com/gitlab-org/cli/pkg/iostreams"

	"gitlab.com/gitlab-org/cli/pkg/tableprinter"
	"gitlab.com/gitlab-org/cli/pkg/utils"

	"github.com/xanzy/go-gitlab"
)

func DisplayAllReleases(io *iostreams.IOStreams, releases []*gitlab.Release, repoName string) string {
	c := io.Color()
	table := tableprinter.NewTablePrinter()
	for _, r := range releases {
		table.AddRow(r.Name, r.TagName, c.Gray(utils.TimeToPrettyTimeAgo(*r.CreatedAt)))
	}

	return table.Render()
}

func RenderReleaseAssertLinks(assets []*gitlab.ReleaseLink) string {
	if len(assets) == 0 {
		return "There are no assets for this release"
	}
	t := tableprinter.NewTablePrinter()
	for _, asset := range assets {
		t.AddRow(asset.Name, asset.DirectAssetURL)
		// assetsPrint += asset.DirectAssetURL + "\n"
	}
	return t.String()
}

func DisplayRelease(io *iostreams.IOStreams, r *gitlab.Release, repo glrepo.Interface) string {
	c := io.Color()
	duration := utils.TimeToPrettyTimeAgo(*r.CreatedAt)
	description, err := utils.RenderMarkdown(r.Description, io.BackgroundColor())
	if err != nil {
		description = r.Description
	}

	var assetsSources string
	for _, asset := range r.Assets.Sources {
		assetsSources += asset.URL + "\n"
	}

	footer := fmt.Sprintf(c.Gray("View this release on GitLab at %s"), r.Links.Self)
	return fmt.Sprintf("%s\n%s released this %s\n%s - %s\n%s\n%s\n%s\n%s\n%s\n\n%s", // whoops
		c.Bold(r.Name), r.Author.Name, duration, r.Commit.ShortID, r.TagName, description, c.Bold("ASSETS"),
		RenderReleaseAssertLinks(r.Assets.Links), c.Bold("SOURCES"), assetsSources, footer,
	)
}

func AssetsFromArgs(args []string) (assets []*upload.ReleaseFile, err error) {
	for _, arg := range args {
		var label string
		var linkType string
		fn := arg
		if arr := strings.SplitN(arg, "#", 3); len(arr) > 0 {
			fn = arr[0]
			if len(arr) > 1 {
				label = arr[1]
			}
			if len(arr) > 2 {
				linkType = arr[2]
			}
		}

		var fi os.FileInfo
		fi, err = os.Stat(fn)
		if err != nil {
			return
		}

		if label == "" {
			label = fi.Name()
		}

		rf := &upload.ReleaseFile{
			Open: func() (io.ReadCloser, error) {
				return os.Open(fn)
			},
			Name:  fi.Name(),
			Label: label,
			Path:  fn,
		}

		// Only add a link type if it was specified
		// Otherwise the GitLab API will default to 'other' if it was omitted
		if linkType != "" {
			linkTypeVal := gitlab.LinkTypeValue(linkType)
			rf.Type = &linkTypeVal
		}

		assets = append(assets, rf)
	}
	return
}

func CreateReleaseAssets(io *iostreams.IOStreams, client *gitlab.Client, assetFiles []*upload.ReleaseFile, assetLinks []*upload.ReleaseAsset, repoName string, tagName string) error {
	if assetFiles == nil && assetLinks == nil {
		return nil
	}

	uploadCtx := upload.Context{
		IO:          io,
		Client:      client,
		AssetsLinks: assetLinks,
		AssetFiles:  assetFiles,
	}

	color := io.Color()
	io.Logf("%s Uploading release assets %s=%s %s=%s\n",
		color.ProgressIcon(),
		color.Blue("repo"), repoName,
		color.Blue("tag"), tagName)

	if err := uploadCtx.UploadFiles(repoName, tagName); err != nil {
		return cmdutils.WrapError(err, "upload failed")
	}

	// create asset link for assets provided as json
	if err := uploadCtx.CreateReleaseAssetLinks(repoName, tagName); err != nil {
		return cmdutils.WrapError(err, "failed to create release link")
	}
	return nil
}
