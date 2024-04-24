package upload

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/xanzy/go-gitlab"
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
