package upload

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/xanzy/go-gitlab"

	"gitlab.com/gitlab-org/cli/commands/cmdtest"
)

func TestReleaseUtilsUpload_AliasFilePathToAssetDirectPath(t *testing.T) {
	tests := []struct {
		name                 string
		givenReleaseAsset    *ReleaseAsset
		expectedReleaseAsset *ReleaseAsset
		expectedAliased      bool
	}{
		{
			name:                 "no filepath, no direct_asset_path",
			givenReleaseAsset:    &ReleaseAsset{Name: gitlab.Ptr("any-name")},
			expectedReleaseAsset: &ReleaseAsset{Name: gitlab.Ptr("any-name")},
			expectedAliased:      false,
		},
		{
			name:                 "no filepath, but direct_asset_path",
			givenReleaseAsset:    &ReleaseAsset{Name: gitlab.Ptr("any-name"), DirectAssetPath: gitlab.Ptr("/any-path")},
			expectedReleaseAsset: &ReleaseAsset{Name: gitlab.Ptr("any-name"), DirectAssetPath: gitlab.Ptr("/any-path")},
			expectedAliased:      false,
		},
		{
			name:                 "filepath, but not direct_asset_path",
			givenReleaseAsset:    &ReleaseAsset{Name: gitlab.Ptr("any-name"), FilePath: gitlab.Ptr("/any-path")},
			expectedReleaseAsset: &ReleaseAsset{Name: gitlab.Ptr("any-name"), DirectAssetPath: gitlab.Ptr("/any-path")},
			expectedAliased:      true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			aliased, err := aliasFilePathToDirectAssetPath(tt.givenReleaseAsset)

			assert.NoError(t, err)
			assert.Equal(t, tt.expectedAliased, aliased)
			assert.Equal(t, tt.expectedReleaseAsset, tt.givenReleaseAsset)
		})
	}
}

func TestReleaseUtilsUpload_AliasFilePathToAssetDirectPath_Conflict(t *testing.T) {
	asset := &ReleaseAsset{
		FilePath:        gitlab.Ptr("/any-path"),
		DirectAssetPath: gitlab.Ptr("/any-path"),
	}

	aliased, err := aliasFilePathToDirectAssetPath(asset)

	target := &ConflictDirectAssetPathError{}
	assert.ErrorAs(t, err, &target)
	assert.False(t, aliased)
}

func TestContext_CreateReleaseAssetFromProjectFile(t *testing.T) {
	expected := &ReleaseAsset{
		Name:            gitlab.Ptr("label"),
		URL:             gitlab.Ptr("https://gitlab.com/-/project/42/uploads/66dbcd21ec5d24ed6ea225176098d52b/test_file.txt"),
		DirectAssetPath: gitlab.Ptr("/test_file.txt"),
		LinkType:        gitlab.Ptr(gitlab.OtherLinkType),
	}

	releaseFile := &ReleaseFile{
		Name:  "test_file.txt",
		Label: "label",
		Path:  "/test_file.txt",
		Type:  gitlab.Ptr(gitlab.OtherLinkType),
	}
	projectFile := &gitlab.ProjectFile{
		FullPath: "/-/project/42/uploads/66dbcd21ec5d24ed6ea225176098d52b/test_file.txt",
	}

	f := cmdtest.StubFactory("https://gitlab.com/cli-automated-testing/test")
	client, _ := f.HttpClient()
	context := &Context{
		Client: client,
	}

	releaseAsset := context.CreateReleaseAssetFromProjectFile(releaseFile, projectFile)
	assert.Equal(t, expected, releaseAsset)
}
