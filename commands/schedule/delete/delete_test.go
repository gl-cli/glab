package delete

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gitlab.com/gitlab-org/cli/commands/cmdtest"
	"gitlab.com/gitlab-org/cli/pkg/httpmock"
	"gitlab.com/gitlab-org/cli/test"
)

func Test_ScheduleDelete(t *testing.T) {
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
			Name:        "Schedule deleted",
			ExpectedMsg: []string{"Deleted schedule with ID 1"},
			cli:         "1",
			httpMocks: []httpMock{
				{
					http.MethodDelete,
					"/api/v4/projects/OWNER/REPO/pipeline_schedules/1",
					http.StatusNoContent,
					"",
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
			if err != nil {
				if tc.wantErr == true {
					if assert.Error(t, err) {
						require.Equal(t, tc.wantStderr, err.Error())
					}
					return
				}
			}

			for _, msg := range tc.ExpectedMsg {
				require.Contains(t, out.String(), msg)
			}
		})
	}
}

func runCommand(rt http.RoundTripper, isTTY bool, cli string) (*test.CmdOut, error) {
	ios, _, stdout, stderr := cmdtest.InitIOStreams(isTTY, "")
	factory := cmdtest.InitFactory(ios, rt)
	_, _ = factory.HttpClient()
	cmd := NewCmdDelete(factory)
	return cmdtest.ExecuteCommand(cmd, cli, stdout, stderr)
}
