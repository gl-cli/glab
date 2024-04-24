package upload

import (
	"fmt"
	"io"

	"github.com/xanzy/go-gitlab"
	"gitlab.com/gitlab-org/cli/pkg/iostreams"
)

// ConflictDirectAssetPathError is returned when both direct_asset_path and the deprecated filepath as specified for an asset link.
type ConflictDirectAssetPathError struct {
	assetLinkName *string
}

func (e *ConflictDirectAssetPathError) Error() string {
	var name string
	if e.assetLinkName != nil {
		name = *e.assetLinkName
	} else {
		name = "(without a name)"
	}
	return fmt.Sprintf("asset link %s contains both `direct_asset_path` and `filepath` (deprecated) fields. Remove the deprecated `filepath` field.", name)
}

type ReleaseAsset struct {
	Name *string `json:"name,omitempty"`
	URL  *string `json:"url,omitempty"`
	// Deprecated FilePath use DirectAssetPath instead.
	FilePath        *string               `json:"filepath,omitempty"`
	DirectAssetPath *string               `json:"direct_asset_path,omitempty"`
	LinkType        *gitlab.LinkTypeValue `json:"link_type,omitempty"`
}

type ReleaseFile struct {
	Open  func() (io.ReadCloser, error)
	Name  string
	Label string
	Path  string
	Type  *gitlab.LinkTypeValue
}

func CreateLink(c *gitlab.Client, projectID, tagName string, asset *ReleaseAsset) (*gitlab.ReleaseLink, bool /* aliased deprecated filepath */, error) {
	aliased, err := aliasFilePathToDirectAssetPath(asset)
	if err != nil {
		return nil, false, err
	}

	releaseLink, _, err := c.ReleaseLinks.CreateReleaseLink(projectID, tagName, &gitlab.CreateReleaseLinkOptions{
		Name:            asset.Name,
		URL:             asset.URL,
		DirectAssetPath: asset.DirectAssetPath,
		LinkType:        asset.LinkType,
	})
	if err != nil {
		return nil, false, err
	}

	return releaseLink, aliased, nil
}

type Context struct {
	Client      *gitlab.Client
	IO          *iostreams.IOStreams
	AssetFiles  []*ReleaseFile
	AssetsLinks []*ReleaseAsset
}

// UploadFiles uploads a file into a release repository.
func (c *Context) UploadFiles(projectID, tagName string) error {
	if c.AssetFiles == nil {
		return nil
	}
	color := c.IO.Color()
	for _, file := range c.AssetFiles {
		fmt.Fprintf(c.IO.StdErr, "%s Uploading to release\t%s=%s %s=%s\n",
			color.ProgressIcon(), color.Blue("file"), file.Path,
			color.Blue("name"), file.Name)

		r, err := file.Open()
		if err != nil {
			return err
		}
		projectFile, _, err := c.Client.Projects.UploadFile(
			projectID,
			r,
			file.Name,
			nil,
		)
		if err != nil {
			return err
		}

		baseURL := c.Client.BaseURL()
		baseURL.Path = "/"

		// projectFile.URL from upload: /uploads/<hash>/filename.txt
		linkURL := baseURL.String() + projectID + projectFile.URL
		filename := "/" + file.Name

		_, _, err = CreateLink(c.Client, projectID, tagName, &ReleaseAsset{
			Name:            &file.Label,
			URL:             &linkURL,
			DirectAssetPath: &filename,
			LinkType:        file.Type,
		})
		if err != nil {
			return err
		}
	}
	c.AssetFiles = nil

	return nil
}

func (c *Context) CreateReleaseAssetLinks(projectID string, tagName string) error {
	if c.AssetsLinks == nil {
		return nil
	}
	color := c.IO.Color()
	for _, asset := range c.AssetsLinks {
		releaseLink, aliased, err := CreateLink(c.Client, projectID, tagName, asset)
		if err != nil {
			return err
		}
		fmt.Fprintf(c.IO.StdErr, "%s Added release asset\t%s=%s %s=%s\n",
			color.GreenCheck(), color.Blue("name"), *asset.Name,
			color.Blue("url"), releaseLink.DirectAssetURL)

		if aliased {
			fmt.Fprintf(c.IO.StdErr, "\t%s Aliased deprecated `filepath` field to `direct_asset_path`. Replace `filepath` with `direct_asset_path`\t%s=%s\n",
				color.WarnIcon(),
				color.Blue("name"), *asset.Name)
		}

	}
	c.AssetsLinks = nil

	return nil
}

// aliasFilePathToDirectAssetPath ensures that the Asset Link uses the direct_asset_path and not the filepath.
// The filepath is deprecated and will be fully removed in GitLab 17.0.
// See https://docs.gitlab.com/ee/update/deprecations.html?removal_milestone=17.0#filepath-field-in-releases-and-release-links-apis
func aliasFilePathToDirectAssetPath(asset *ReleaseAsset) (bool /* aliased */, error) {
	if asset.FilePath == nil || *asset.FilePath == "" {
		// There is no deprecated filepath set, so we are good.
		return false, nil
	}

	if asset.DirectAssetPath != nil && *asset.DirectAssetPath != "" {
		// Both, filepath and direct_asset_path are set, so we have a conflict
		return false, &ConflictDirectAssetPathError{asset.Name}
	}

	// We use the set filepath as direct_asset_path
	// and clear the filepath to net send it via API.
	asset.DirectAssetPath = asset.FilePath
	asset.FilePath = nil

	return true, nil
}
