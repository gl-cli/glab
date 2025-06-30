package create

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gitlab.com/gitlab-org/cli/internal/httpmock"
	"gitlab.com/gitlab-org/cli/internal/testing/cmdtest"
	"gitlab.com/gitlab-org/cli/test"
)

func Test_LabelCreate(t *testing.T) {
	type httpMock struct {
		method string
		path   string
		status int
		body   string
	}

	testCases := []struct {
		Name        string
		ExpectedMsg []string
		wantErr     bool
		cli         string
		wantStderr  string
		httpMocks   []httpMock
	}{
		{
			Name:        "Label created",
			ExpectedMsg: []string{"Created label: foo\nWith color: #FF0000"},
			cli:         "--name foo --color red",
			httpMocks: []httpMock{
				{
					http.MethodPost,
					"/api/v4/projects/OWNER/REPO/labels",
					http.StatusCreated,
					`{"name":"foo","color":"#FF0000"}`,
				},
			},
		},
		{
			Name:        "Label not created because of missing name",
			wantStderr:  "required flag(s) \"name\" not set",
			wantErr:     true,
			ExpectedMsg: []string{""},
		},
		{
			Name:        "Label created with description",
			ExpectedMsg: []string{"Created label: foo\nWith color: #FF0000"},
			cli:         "--name foo --color red --description foo_desc",
			httpMocks: []httpMock{
				{
					http.MethodPost,
					"/api/v4/projects/OWNER/REPO/labels",
					http.StatusCreated,
					`{"name":"foo","color":"#FF0000", "description":"foo_desc"}`,
				},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.Name, func(t *testing.T) {
			fakeHTTP := &httpmock.Mocker{
				MatchURL: httpmock.PathAndQuerystring,
			}
			defer fakeHTTP.Verify(t)

			for _, mock := range tc.httpMocks {
				fakeHTTP.RegisterResponder(mock.method, mock.path, httpmock.NewStringResponse(mock.status, mock.body))
			}

			out, err := runCommand(fakeHTTP, tc.cli)

			for _, msg := range tc.ExpectedMsg {
				require.Contains(t, out.String(), msg)
			}
			if err != nil {
				if tc.wantErr == true {
					if assert.Error(t, err) {
						require.Equal(t, tc.wantStderr, err.Error())
					}
					return
				}
			}
		})
	}
}

func runCommand(rt http.RoundTripper, cli string) (*test.CmdOut, error) {
	ios, _, stdout, stderr := cmdtest.TestIOStreams()
	factory := cmdtest.InitFactory(ios, rt)
	cmd := NewCmdCreate(factory)
	return cmdtest.ExecuteCommand(cmd, cli, stdout, stderr)
}
