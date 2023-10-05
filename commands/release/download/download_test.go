package download

import (
	"net/http"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gitlab.com/gitlab-org/cli/api"
	"gitlab.com/gitlab-org/cli/commands/release/releaseutils/upload"
	"gitlab.com/gitlab-org/cli/pkg/httpmock"
	"gitlab.com/gitlab-org/cli/pkg/iostreams"
)

func doesFileExist(fileName string) bool {
	_, err := os.Stat(fileName)
	return err == nil
}

func Test_downloadAssets(t *testing.T) {
	assetUrl := "https://gitlab.com/gitlab-org/cli/-/archive/"

	fakeHTTP := &httpmock.Mocker{
		MatchURL: httpmock.HostAndPath,
	}

	client, _ := api.TestClient(&http.Client{Transport: fakeHTTP}, "", "", false)

	tests := []struct {
		name     string
		filename string
		want     string
		wantErr  bool
	}{
		{
			name:     "A regular filename",
			filename: "cli-v1.22.0.tar",
			want:     "cli-v1.22.0.tar",
		},
		{
			name:     "A filename with directory traversal",
			filename: "cli-v1.../../22.0.tar",
			want:     "22.0.tar",
		},
		{
			name:     "A particularly nasty filename",
			filename: "..././..././..././etc/password_file",
			wantErr:  true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fullUrl := assetUrl + tt.filename
			fakeHTTP.RegisterResponder(http.MethodGet, fullUrl, httpmock.NewStringResponse(http.StatusOK, `test_data`))

			io, _, _, _ := iostreams.Test()

			release := &upload.ReleaseAsset{
				Name: &tt.filename,
				URL:  &fullUrl,
			}

			releases := []*upload.ReleaseAsset{release}

			tempPath, tempPathErr := os.MkdirTemp("/tmp", "temp_tester")
			require.NoError(t, tempPathErr)

			filePathWanted := filepath.Join(tempPath, tt.want)

			err := downloadAssets(client, io, releases, tempPath)

			if tt.wantErr {
				assert.Error(t, err, "Should error out if a path doesn't exist")
				return
			}

			assert.NoError(t, err, "Should not have errors")
			assert.True(t, doesFileExist(filePathWanted), "File should exist")
		})
	}
}
