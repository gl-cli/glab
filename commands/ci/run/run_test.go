package run

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os/exec"
	"testing"

	"gitlab.com/gitlab-org/cli/commands/cmdtest"
	"gitlab.com/gitlab-org/cli/internal/run"

	"github.com/stretchr/testify/assert"
	"gitlab.com/gitlab-org/cli/pkg/httpmock"
	"gitlab.com/gitlab-org/cli/test"
)

type ResponseJSON struct {
	Ref string `json:"ref"`
}

func runCommand(rt http.RoundTripper, cli string) (*test.CmdOut, error, func()) {
	ios, _, stdout, stderr := cmdtest.InitIOStreams(true, "")

	factory := cmdtest.InitFactory(ios, rt)

	factory.Branch = func() (string, error) {
		return "custom-branch-123", nil
	}

	restoreCmd := run.SetPrepareCmd(func(cmd *exec.Cmd) run.Runnable {
		return &test.OutputStub{}
	})

	_, _ = factory.HttpClient()

	cmd := NewCmdRun(factory)
	cmdOut, err := cmdtest.ExecuteCommand(cmd, cli, stdout, stderr)

	return cmdOut, err, restoreCmd
}

func TestCIRun(t *testing.T) {
	tests := []struct {
		name string
		cli  string

		expectedPOSTBody string
		expectedOut      string
		expectedErr      string
	}{
		{
			name:             "when running `ci run` without any parameter, defaults to current branch",
			cli:              "",
			expectedPOSTBody: `"ref":"custom-branch-123"`,
			expectedOut:      "Created pipeline (id: 123), status: created, ref: custom-branch-123, weburl: https://gitlab.com/OWNER/REPO/-/pipelines/123\n",
		},
		{
			name:             "when running `ci run` with branch parameter, run CI at branch",
			cli:              "-b ci-cd-improvement-399",
			expectedPOSTBody: `"ref":"ci-cd-improvement-399"`,
			expectedOut:      "Created pipeline (id: 123), status: created, ref: ci-cd-improvement-399, weburl: https://gitlab.com/OWNER/REPO/-/pipelines/123\n",
		},
		{
			name:             "when running `ci run` with --web opens the browser",
			cli:              "-b web-branch --web",
			expectedPOSTBody: `"ref":"web-branch"`,
			expectedErr:      "Opening gitlab.com/OWNER/REPO/-/pipelines/123 in your browser.\n",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			fakeHTTP := &httpmock.Mocker{
				MatchURL: httpmock.PathAndQuerystring,
			}
			defer fakeHTTP.Verify(t)

			fakeHTTP.RegisterResponder(http.MethodPost, "/api/v4/projects/OWNER/REPO/pipeline",
				func(req *http.Request) (*http.Response, error) {
					rb, _ := io.ReadAll(req.Body)

					var response ResponseJSON
					err := json.Unmarshal(rb, &response)
					if err != nil {
						fmt.Printf("Error when parsing response body %s\n", rb)
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

			output, err, restoreCmd := runCommand(fakeHTTP, tc.cli)
			defer restoreCmd()

			assert.NoErrorf(t, err, "error running command `ci run %s`: %v", tc.cli, err)

			assert.Equal(t, tc.expectedOut, output.String())
			assert.Equal(t, tc.expectedErr, output.Stderr())
		})
	}
}
