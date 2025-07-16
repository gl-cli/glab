package upload

import (
	"bytes"
	"io"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	gitlab "gitlab.com/gitlab-org/api/client-go"
	gitlabtesting "gitlab.com/gitlab-org/api/client-go/testing"
	"gitlab.com/gitlab-org/cli/internal/glinstance"
	"gitlab.com/gitlab-org/cli/internal/testing/cmdtest"
	"go.uber.org/mock/gomock"
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

func TestReleaseUtilsUpload_UploadFiles_ProjectMarkdownFiles(t *testing.T) {
	t.Parallel()

	// GIVEN
	ios, _, _, _ := cmdtest.TestIOStreams()
	tc := gitlabtesting.NewTestClient(t, gitlab.WithBaseURL(glinstance.DefaultHostname))
	testReleaseFile := bytes.NewBufferString("Hello World")
	uploadCtx := &Context{
		Client: tc.Client,
		IO:     ios,
		AssetFiles: []*ReleaseFile{
			{
				Open: func() (io.ReadCloser, error) {
					return io.NopCloser(testReleaseFile), nil
				},
				Name: "test-release-file.txt",
				Path: "./foobar/test-release-file.txt",
				Type: gitlab.Ptr(gitlab.OtherLinkType),
			},
		},
		AssetsLinks: []*ReleaseAsset{},
	}

	// setup mock expections
	gomock.InOrder(
		tc.MockProjectMarkdownUploads.EXPECT().
			UploadProjectMarkdown("any-project", gomock.Any(), "test-release-file.txt", gomock.Any()).
			Return(&gitlab.MarkdownUploadedFile{FullPath: "test-release-file.txt"}, nil, nil),
		tc.MockReleaseLinks.EXPECT().CreateReleaseLink("any-project", "42.0.0", gomock.Any()),
	)

	// WHEN
	err := uploadCtx.UploadFiles("any-project", "42.0.0", false)

	// THEN
	require.NoError(t, err)
}

func TestReleaseUtilsUpload_UploadFiles_GenericPackageRegistry(t *testing.T) {
	t.Parallel()

	// GIVEN
	ios, _, _, _ := cmdtest.TestIOStreams()
	tc := gitlabtesting.NewTestClient(t, gitlab.WithBaseURL(glinstance.DefaultHostname))
	testReleaseFile := bytes.NewBufferString("Hello World")
	uploadCtx := &Context{
		Client: tc.Client,
		IO:     ios,
		AssetFiles: []*ReleaseFile{
			{
				Open: func() (io.ReadCloser, error) {
					return io.NopCloser(testReleaseFile), nil
				},
				Name: "test-release-file.txt",
				Path: "./foobar/test-release-file.txt",
				Type: gitlab.Ptr(gitlab.OtherLinkType),
			},
		},
		AssetsLinks: []*ReleaseAsset{},
	}

	// setup mock expections
	gomock.InOrder(
		tc.MockGenericPackages.EXPECT().
			PublishPackageFile("any-project", releasePackageName, "42.0.0", "test-release-file.txt", gomock.Any(), nil),
		tc.MockGenericPackages.EXPECT().
			FormatPackageURL("any-project", releasePackageName, "42.0.0", "test-release-file.txt"),
		tc.MockReleaseLinks.EXPECT().CreateReleaseLink("any-project", "42.0.0", gomock.Any()),
	)

	// WHEN
	err := uploadCtx.UploadFiles("any-project", "42.0.0", true)

	// THEN
	require.NoError(t, err)
}
