package update

import (
	"net/http"
	"testing"

	"gitlab.com/gitlab-org/cli/commands/cmdtest"
	"gitlab.com/gitlab-org/cli/test"

	"github.com/stretchr/testify/assert"
	"gitlab.com/gitlab-org/cli/pkg/httpmock"
)

func runCommand(rt http.RoundTripper, version string) (*test.CmdOut, error) {
	ios, _, stdout, stderr := cmdtest.InitIOStreams(true, "")
	factory := cmdtest.InitFactory(ios, rt)
	_, _ = factory.HttpClient()

	cmd := NewCheckUpdateCmd(factory, version)

	return cmdtest.ExecuteCommand(cmd, "", stdout, stderr)
}

func TestNewCheckUpdateCmd(t *testing.T) {
	type args struct {
		version string
	}
	tests := []struct {
		name   string
		args   args
		stdOut string
		stdErr string
	}{
		{
			name: "same version",
			args: args{
				version: "v1.11.1",
			},
			stdErr: "You are already using the latest version of glab!\n",
		},
		{
			name: "older version",
			args: args{
				version: "v1.11.0",
			},
			stdErr: "A new version of glab has been released: v1.11.0 -> v1.11.1\nhttps://gitlab.com/gitlab-org/cli/-/releases/v1.11.1\n",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fakeHTTP := &httpmock.Mocker{
				MatchURL: httpmock.PathAndQuerystring,
			}
			defer fakeHTTP.Verify(t)

			fakeHTTP.RegisterResponder(http.MethodGet, `https://gitlab.com/api/v4/projects/gitlab-org/cli/releases?page=1&per_page=1`,
				func(req *http.Request) (*http.Response, error) {
					// Ensure no token is sent when checking for a glab update
					assert.Empty(t, req.Header.Get("Private-Token"))

					resp, _ := httpmock.NewStringResponse(http.StatusOK, `[{
							"tag_name": "v1.11.1",
							"name": "v1.11.1",
							"created_at": "2020-11-03T05:33:29Z",
							"released_at": "2020-11-03T05:39:04Z"
						}]`)(req)

					return resp, nil
				},
			)

			output, err := runCommand(fakeHTTP, tt.args.version)

			assert.Nil(t, err)
			assert.Empty(t, output.String())
			assert.Equal(t, tt.stdErr, output.Stderr())
		})
	}
}

func TestNewCheckUpdateCmd_error(t *testing.T) {
	fakeHTTP := &httpmock.Mocker{
		MatchURL: httpmock.PathAndQuerystring,
	}
	defer fakeHTTP.Verify(t)

	fakeHTTP.RegisterResponder(http.MethodGet, `https://gitlab.com/api/v4/projects/gitlab-org/cli/releases?page=1&per_page=1`,
		httpmock.NewStringResponse(http.StatusNotFound, `
				{
					"message": "test error"
				}
			`))

	output, err := runCommand(fakeHTTP, "1.11.0")

	assert.NotNil(t, err)
	assert.Equal(t, `failed checking for glab updates: 404 Not Found`, err.Error())
	assert.Equal(t, "", output.String())
	assert.Equal(t, "", output.Stderr())
}

func TestNewCheckUpdateCmd_no_release(t *testing.T) {
	fakeHTTP := &httpmock.Mocker{
		MatchURL: httpmock.PathAndQuerystring,
	}
	defer fakeHTTP.Verify(t)

	fakeHTTP.RegisterResponder(http.MethodGet, `https://gitlab.com/api/v4/projects/gitlab-org/cli/releases?page=1&per_page=1`,
		httpmock.NewStringResponse(http.StatusOK, `[]`))

	output, err := runCommand(fakeHTTP, "1.11.0")

	assert.NotNil(t, err)
	assert.Equal(t, "no release found for glab.", err.Error())
	assert.Equal(t, "", output.String())
	assert.Equal(t, "", output.Stderr())
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

func TestCheckUpdate_NoRun(t *testing.T) {
	tests := []struct {
		name            string
		previousCommand string
	}{
		{
			name:            "when previous command is check-update",
			previousCommand: "check-update",
		},
		{
			name:            "when previous command is an alias for check-update",
			previousCommand: "update",
		},
		{
			name:            "when previous command is completion",
			previousCommand: "completion",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Nil(t, CheckUpdate(nil, "1.1.1", true, tt.previousCommand))
		})
	}
}
