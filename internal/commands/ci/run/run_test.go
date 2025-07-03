package run

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os/exec"
	"testing"

	gitlab "gitlab.com/gitlab-org/api/client-go"
	"gitlab.com/gitlab-org/cli/internal/api"
	"gitlab.com/gitlab-org/cli/internal/glinstance"
	"gitlab.com/gitlab-org/cli/internal/prompt"

	"gitlab.com/gitlab-org/cli/internal/run"
	"gitlab.com/gitlab-org/cli/internal/testing/cmdtest"

	"github.com/stretchr/testify/assert"
	"gitlab.com/gitlab-org/cli/internal/testing/httpmock"
	"gitlab.com/gitlab-org/cli/test"
)

type ResponseJSON struct {
	Ref string `json:"ref"`
}

func runCommand(t *testing.T, rt http.RoundTripper, cli string) (*test.CmdOut, error, func()) {
	ios, _, stdout, stderr := cmdtest.TestIOStreams(cmdtest.WithTestIOStreamsAsTTY(true))
	factory := cmdtest.NewTestFactory(ios,
		cmdtest.WithGitLabClient(cmdtest.NewTestApiClient(t, &http.Client{Transport: rt}, "", glinstance.DefaultHostname).Lab()),
	)

	factory.BranchStub = func() (string, error) {
		return "custom-branch-123", nil
	}

	restoreCmd := run.SetPrepareCmd(func(cmd *exec.Cmd) run.Runnable {
		return &test.OutputStub{}
	})

	cmd := NewCmdRun(factory)
	cmdOut, err := cmdtest.ExecuteCommand(cmd, cli, stdout, stderr)

	return cmdOut, err, restoreCmd
}

func TestCIRun(t *testing.T) {
	t.Parallel()

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

			output, err, restoreCmd := runCommand(t, fakeHTTP, tc.cli)
			defer restoreCmd()

			assert.NoErrorf(t, err, "error running command `ci run %s`: %v", tc.cli, err)

			assert.Equal(t, tc.expectedOut, output.String())
			assert.Equal(t, tc.expectedErr, output.Stderr())
		})
	}
}

func TestCIRunMrPipeline(t *testing.T) {
	tests := []struct {
		name string
		cli  string

		expectedOut string
		expectedErr string
		mrIid       int
	}{
		{
			name:        "bare mr flag",
			cli:         "--mr",
			expectedOut: "Created pipeline (id: 21370), status: created, ref: , weburl: https://gitlab.com/OWNER/REPO/-/pipelines/21370\n",
			mrIid:       2137,
		},
		{
			name:        "mr flag with branch specified",
			cli:         "--mr --branch branchy",
			expectedOut: "Created pipeline (id: 7350), status: created, ref: , weburl: https://gitlab.com/OWNER/REPO/-/pipelines/7350\n",
			mrIid:       735,
		},
		{
			name:        "mr flag with branch specified & multiple MRs",
			cli:         "--mr --branch my_branch_with_a_myriad_of_mrs",
			expectedOut: "Created pipeline (id: 12340), status: created, ref: , weburl: https://gitlab.com/OWNER/REPO/-/pipelines/12340\n",
			mrIid:       1234,
		},
		{
			name:        "mr flag with branch specified & no MRs",
			cli:         "--mr --branch branch_without_mrs",
			expectedErr: "error running command `ci run --mr --branch branch_without_mrs`: no open merge request available for \"branch_without_mrs\"",
			mrIid:       1234,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			fakeHTTP := &httpmock.Mocker{
				MatchURL: httpmock.PathAndQuerystring,
			}
			defer fakeHTTP.Verify(t)
			if tc.expectedErr == "" {
				iid := tc.mrIid
				fakeHTTP.RegisterResponder(http.MethodPost, "/api/v4/projects/OWNER/REPO/merge_requests/"+fmt.Sprint(iid)+"/pipelines",
					func(req *http.Request) (*http.Response, error) {
						pipelineId := iid * 10
						resp, _ := httpmock.NewStringResponse(http.StatusOK, fmt.Sprintf(`{
 						"id": %d,
 						"status": "created",
			            "web_url": "https://gitlab.com/OWNER/REPO/-/pipelines/%d"}`, pipelineId, pipelineId))(req)
						return resp, nil
					},
				)
			}
			api.ListMRs = func(client *gitlab.Client, projectID interface{}, opts *gitlab.ListProjectMergeRequestsOptions, listOpts ...api.CliListMROption) ([]*gitlab.BasicMergeRequest, error) {
				if *opts.SourceBranch == "custom-branch-123" {
					return []*gitlab.BasicMergeRequest{
						{
							IID: 2137,
							Author: &gitlab.BasicUser{
								Username: "Huan Pablo Secundo",
							},
						},
					}, nil
				}
				if *opts.SourceBranch == "branchy" {
					return []*gitlab.BasicMergeRequest{
						{
							IID: 735,
							Author: &gitlab.BasicUser{
								Username: "Franciszek",
							},
						},
					}, nil
				}
				if *opts.SourceBranch == "my_branch_with_a_myriad_of_mrs" {
					return []*gitlab.BasicMergeRequest{
						{
							IID: 1234,
							Author: &gitlab.BasicUser{
								Username: "Chris Harms",
							},
						},
						{
							IID: 666,
							Author: &gitlab.BasicUser{
								Username: "Bruce Dickinson",
							},
						},
					}, nil
				}
				if *opts.SourceBranch == "branch_without_mrs" {
					return []*gitlab.BasicMergeRequest{}, nil
				}
				return nil, fmt.Errorf("unexpected branch in this mock :(")
			}
			as, restoreAsk := prompt.InitAskStubber()
			defer restoreAsk()

			as.Stub([]*prompt.QuestionStub{
				{
					Name:  "mr",
					Value: "!1234 (my_branch_with_a_myriad_of_mrs) by @Chris Harms",
				},
			})
			output, err, restoreCmd := runCommand(t, fakeHTTP, tc.cli)
			defer restoreCmd()

			if tc.expectedErr == "" {
				assert.NoErrorf(t, err, "error running command `ci run %s`: %v", tc.cli, err)

				assert.Equal(t, tc.expectedOut, output.String())
				assert.Equal(t, tc.expectedErr, output.Stderr())
			} else {
				assert.Errorf(t, err, "error running command `ci run %s`: %v", tc.cli, err)
			}
		})
	}
}
