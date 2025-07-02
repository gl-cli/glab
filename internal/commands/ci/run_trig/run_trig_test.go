package run_trig

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"testing"

	"gitlab.com/gitlab-org/cli/internal/glinstance"
	"gitlab.com/gitlab-org/cli/internal/testing/cmdtest"

	"github.com/stretchr/testify/assert"
	"gitlab.com/gitlab-org/cli/internal/testing/httpmock"
	"gitlab.com/gitlab-org/cli/test"
)

type ResponseJSON struct {
	Token string `json:"token"`
	Ref   string `json:"ref"`
}

func runCommand(t *testing.T, rt http.RoundTripper, cli string) (*test.CmdOut, error) {
	ios, _, stdout, stderr := cmdtest.TestIOStreams()
	factory := cmdtest.NewTestFactory(ios,
		cmdtest.WithGitLabClient(cmdtest.NewTestApiClient(t, &http.Client{Transport: rt}, "", glinstance.DefaultHostname).Lab()),
	)

	factory.BranchStub = func() (string, error) {
		return "custom-branch-123", nil
	}

	cmd := NewCmdRunTrig(factory)

	return cmdtest.ExecuteCommand(cmd, cli, stdout, stderr)
}

func TestCIRun(t *testing.T) {
	tests := []struct {
		name             string
		cli              string
		ciJobToken       string
		expectedPOSTBody string
		expectedOut      string
	}{
		{
			name:             "when running `ci run-trig` without branch parameter, defaults to current branch",
			cli:              "-t foobar",
			ciJobToken:       "",
			expectedPOSTBody: `"ref":"custom-branch-123"`,
			expectedOut:      "Created pipeline (ID: 123), status: created, ref: custom-branch-123, weburl: https://gitlab.com/OWNER/REPO/-/pipelines/123\n",
		},
		{
			name:             "when running `ci run-trig` with branch parameter, run CI at branch",
			cli:              "-t foobar -b ci-cd-improvement-399",
			ciJobToken:       "",
			expectedPOSTBody: `"ref":"ci-cd-improvement-399"`,
			expectedOut:      "Created pipeline (ID: 123), status: created, ref: ci-cd-improvement-399, weburl: https://gitlab.com/OWNER/REPO/-/pipelines/123\n",
		},
		{
			name:             "when running `ci run-trig` without any parameter, takes trigger token from env variable",
			cli:              "",
			ciJobToken:       "foobar",
			expectedPOSTBody: `"ref":"custom-branch-123"`,
			expectedOut:      "Created pipeline (ID: 123), status: created, ref: custom-branch-123, weburl: https://gitlab.com/OWNER/REPO/-/pipelines/123\n",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			fakeHTTP := &httpmock.Mocker{
				MatchURL: httpmock.PathAndQuerystring,
			}
			defer fakeHTTP.Verify(t)

			t.Setenv("CI_JOB_TOKEN", tc.ciJobToken)

			fakeHTTP.RegisterResponder(http.MethodPost, "/api/v4/projects/OWNER/REPO/trigger/pipeline",
				func(req *http.Request) (*http.Response, error) {
					rb, _ := io.ReadAll(req.Body)

					var response ResponseJSON
					err := json.Unmarshal(rb, &response)
					if err != nil {
						fmt.Printf("Error when parsing response body %s\n", rb)
					}

					if response.Token != "foobar" {
						fmt.Printf("Invalid token %s\n", rb)
					}

					// ensure CLI runs CI on correct branch
					assert.Contains(t, string(rb), tc.expectedPOSTBody)
					resp, _ := httpmock.NewStringResponse(http.StatusOK, fmt.Sprintf(`{
 						"id": 123,
 						"iid": 123,
 						"project_id": 3,
 						"status": "created",
 						"ref": "%s",
						"web_url": "https://gitlab.com/OWNER/REPO/-/pipelines/123"}`, response.Ref))(req)
					return resp, nil
				},
			)

			output, _ := runCommand(t, fakeHTTP, tc.cli)

			out := output.String()

			assert.Equal(t, tc.expectedOut, out)
			assert.Empty(t, output.Stderr())
		})
	}
}
