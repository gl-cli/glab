package update

import (
	"bytes"
	"fmt"
	"net/http"
	"testing"

	"github.com/profclems/glab/api"
	"github.com/profclems/glab/commands/cmdutils"
	"github.com/profclems/glab/pkg/iostreams"

	"github.com/alecthomas/assert"
	"github.com/jarcoal/httpmock"
	"github.com/xanzy/go-gitlab"
)

func TestNewCheckUpdateCmd(t *testing.T) {
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()

	httpmock.RegisterResponder("GET", `https://gitlab.com/api/v4/projects/gitlab-org%2Fcli/releases`,
		httpmock.NewStringResponder(200, `[{"tag_name": "v1.11.1",
  "name": "v1.11.1",
  "created_at": "2020-11-03T05:33:29Z",
  "released_at": "2020-11-03T05:39:04Z"}]`))

	factory, _, stdout, stderr, err := makeTestFactory()
	assert.Nil(t, err)

	type args struct {
		version string
	}
	tests := []struct {
		name    string
		args    args
		stdOut  string
		stdErr  string
		wantErr bool
	}{
		{
			name: "same version",
			args: args{
				version: "v1.11.1",
			},
			stdOut: "✓ You are already using the latest version of glab\n",
			stdErr: "",
		},
		{
			name: "older version",
			args: args{
				version: "v1.11.0",
			},
			stdOut: "A new version of glab has been released: v1.11.0 → v1.11.1\nhttps://gitlab.com/gitlab-org/cli/-/releases/v1.11.1\n",
			stdErr: "",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := NewCheckUpdateCmd(factory, tt.args.version).Execute()
			if tt.wantErr {
				assert.Nil(t, err)
			}

			assert.Equal(t, tt.stdOut, stdout.String())
			assert.Equal(t, tt.stdErr, stderr.String())

			// clean up
			stdout.Reset()
			stderr.Reset()
		})
	}
}

func TestNewCheckUpdateCmd_error(t *testing.T) {
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()

	httpmock.RegisterResponder("GET", `https://gitlab.com/api/v4/projects/gitlab-org%2Fcli/releases`,
		httpmock.NewErrorResponder(fmt.Errorf("an error expected")))

	factory, _, stdout, stderr, err := makeTestFactory()
	assert.Nil(t, err)

	err = NewCheckUpdateCmd(factory, "1.11.0").Execute()
	assert.NotNil(t, err)
	assert.Equal(t, "could not check for update: Get \"https://gitlab.com/api/v4/projects/gitlab-org%2Fcli/releases?page=1&per_page=1\": an error expected", err.Error())
	assert.Equal(t, "", stdout.String())
	assert.Equal(t, "", stderr.String())
}

func TestNewCheckUpdateCmd_no_release(t *testing.T) {
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()

	httpmock.RegisterResponder("GET", `https://gitlab.com/api/v4/projects/gitlab-org%2Fcli/releases`,
		httpmock.NewStringResponder(200, `[]`))

	factory, _, stdout, stderr, err := makeTestFactory()
	assert.Nil(t, err)

	err = NewCheckUpdateCmd(factory, "1.11.0").Execute()
	assert.NotNil(t, err)
	assert.Equal(t, "no release found for glab", err.Error())
	assert.Equal(t, "", stdout.String())
	assert.Equal(t, "", stderr.String())
}

func Test_isOlderVersion(t *testing.T) {
	type args struct {
		latestVersion  string
		currentVersion string
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			name: "latest is newer",
			args: args{"v1.10.0", "v1.9.1"},
			want: true,
		},
		{
			name: "latest is current",
			args: args{"v1.9.2", "v1.9.2"},
			want: false,
		},
		{
			name: "latest is older",
			args: args{"v1.9.0", "v1.9.2-pre.1"},
			want: false,
		},
		{
			name: "current is prerelease",
			args: args{"v1.9.0", "v1.9.0-pre.1"},
			want: true,
		},
		{
			name: "latest is older (against prerelease)",
			args: args{"v1.9.0", "v1.10.0-pre.1"},
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isOlderVersion(tt.args.latestVersion, tt.args.currentVersion); got != tt.want {
				t.Errorf("isOlderVersion(%s, %s) = %v, want %v",
					tt.args.latestVersion, tt.args.currentVersion, got, tt.want)
			}
		})
	}
}

func makeTestFactory() (factory *cmdutils.Factory, in *bytes.Buffer, out *bytes.Buffer, errOut *bytes.Buffer, err error) {
	var apiClient *api.Client
	apiClient, err = api.TestClient(http.DefaultClient, "", "gitlab.com", false)
	if err != nil {
		return
	}

	factory = cmdutils.NewFactory()
	factory.HttpClient = func() (*gitlab.Client, error) {
		return apiClient.Lab(), nil
	}
	factory.IO, _, out, errOut = iostreams.Test()
	return
}
