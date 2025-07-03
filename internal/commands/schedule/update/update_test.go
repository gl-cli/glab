package update

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gitlab.com/gitlab-org/cli/internal/testing/cmdtest"
	"gitlab.com/gitlab-org/cli/internal/testing/httpmock"
	"gitlab.com/gitlab-org/cli/test"
)

func Test_ScheduleEdit(t *testing.T) {
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
			Name:        "Schedule updated",
			ExpectedMsg: []string{"Updated schedule with ID 1"},
			cli:         "1 --cron '*0 * * * *' --description 'example pipeline' --ref 'main'",
			httpMocks: []httpMock{
				{
					http.MethodPut,
					"/api/v4/projects/OWNER/REPO/pipeline_schedules/1",
					http.StatusOK,
					`{"id": 1}`,
				},
			},
		},
		{
			Name:        "Schedule updated with new variable",
			ExpectedMsg: []string{"Updated schedule with ID 1"},
			cli:         "1 --description 'example pipeline' --create-variable 'foo:bar'",
			httpMocks: []httpMock{
				{
					http.MethodPut,
					"/api/v4/projects/OWNER/REPO/pipeline_schedules/1",
					http.StatusOK,
					`{"id": 1}`,
				},
				{
					http.MethodPost,
					"/api/v4/projects/OWNER/REPO/pipeline_schedules/1/variables",
					http.StatusCreated,
					`{}`,
				},
			},
		},
		{
			Name:        "Schedule updated with updated variable",
			ExpectedMsg: []string{"Updated schedule with ID 1"},
			cli:         "1 --description 'example pipeline' --update-variable 'foo:bar'",
			httpMocks: []httpMock{
				{
					http.MethodPut,
					"/api/v4/projects/OWNER/REPO/pipeline_schedules/1",
					http.StatusOK,
					`{"id": 1}`,
				},
				{
					http.MethodPut,
					"/api/v4/projects/OWNER/REPO/pipeline_schedules/1/variables/foo",
					http.StatusOK,
					`{}`,
				},
			},
		},
		{
			Name:        "Schedule updated with deleted variable",
			ExpectedMsg: []string{"Updated schedule with ID 1"},
			cli:         "1 --description 'example pipeline' --delete-variable 'foo'",
			httpMocks: []httpMock{
				{
					http.MethodPut,
					"/api/v4/projects/OWNER/REPO/pipeline_schedules/1",
					http.StatusOK,
					`{"id": 1}`,
				},
				{
					http.MethodDelete,
					"/api/v4/projects/OWNER/REPO/pipeline_schedules/1/variables/foo",
					http.StatusOK,
					`{}`,
				},
			},
		},
		{
			Name:       "Schedule updated with invalid variable format - create",
			cli:        "1 --create-variable 'foo:bar' --create-variable 'foo'",
			wantErr:    true,
			wantStderr: `Invalid format for --create-variable: foo`,
			httpMocks:  []httpMock{},
		},
		{
			Name:       "Schedule updated with invalid variable format - update",
			cli:        "1 --update-variable 'foo:bar' --update-variable 'foo'",
			wantErr:    true,
			wantStderr: `Invalid format for --update-variable: foo`,
			httpMocks:  []httpMock{},
		},
		{
			Name:        "Schedule not changed if no flags are set",
			ExpectedMsg: []string{"Updated schedule with ID 1"},
			cli:         "1",
			httpMocks:   []httpMock{},
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

			out, err := runCommand(t, fakeHTTP, tc.cli)

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

func runCommand(t *testing.T, rt http.RoundTripper, cli string) (*test.CmdOut, error) {
	ios, _, stdout, stderr := cmdtest.TestIOStreams()
	factory := cmdtest.NewTestFactory(ios,
		cmdtest.WithGitLabClient(cmdtest.NewTestApiClient(t, &http.Client{Transport: rt}, "", "gitlab.com").Lab()),
	)
	cmd := NewCmdUpdate(factory)
	return cmdtest.ExecuteCommand(cmd, cli, stdout, stderr)
}
