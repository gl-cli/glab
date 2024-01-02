package create

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gitlab.com/gitlab-org/cli/commands/cmdtest"
	"gitlab.com/gitlab-org/cli/pkg/httpmock"
	"gitlab.com/gitlab-org/cli/test"
)

func Test_ScheduleCreate(t *testing.T) {
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
			Name:        "Schedule created",
			ExpectedMsg: []string{"Created schedule"},
			cli:         "--cron '*0 * * * *' --description 'example pipeline' --ref 'main'",
			httpMocks: []httpMock{
				{
					http.MethodPost,
					"/api/v4/projects/OWNER/REPO/pipeline_schedules",
					http.StatusCreated,
					`{}`,
				},
			},
		},
		{
			Name:        "Schedule not created because of missing ref",
			wantStderr:  "required flag(s) \"ref\" not set",
			wantErr:     true,
			ExpectedMsg: []string{""},
			cli:         "--cron '*0 * * * *' --description 'example pipeline'",
		},
		{
			Name:       "Schedule created but with skipped variable",
			wantStderr: "Invalid format for --variable: foo",
			wantErr:    true,
			cli:        "--cron '*0 * * * *' --description 'example pipeline' --ref 'main'  --variable 'foo'",
			httpMocks: []httpMock{
				{
					http.MethodPost,
					"/api/v4/projects/OWNER/REPO/pipeline_schedules",
					http.StatusCreated,
					`{}`,
				},
			},
		},
		{
			Name:        "Schedule created with variable",
			ExpectedMsg: []string{"Created schedule"},
			cli:         "--cron '*0 * * * *' --description 'example pipeline' --ref 'main' --variable 'foo:bar'",
			httpMocks: []httpMock{
				{
					http.MethodPost,
					"/api/v4/projects/OWNER/REPO/pipeline_schedules",
					http.StatusCreated,
					`{}`,
				},
				{
					http.MethodPost,
					"/api/v4/projects/OWNER/REPO/pipeline_schedules/0/variables",
					http.StatusCreated,
					`{}`,
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

			out, err := runCommand(fakeHTTP, false, tc.cli)

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

func runCommand(rt http.RoundTripper, isTTY bool, cli string) (*test.CmdOut, error) {
	ios, _, stdout, stderr := cmdtest.InitIOStreams(isTTY, "")
	factory := cmdtest.InitFactory(ios, rt)
	_, _ = factory.HttpClient()
	cmd := NewCmdCreate(factory)
	return cmdtest.ExecuteCommand(cmd, cli, stdout, stderr)
}
