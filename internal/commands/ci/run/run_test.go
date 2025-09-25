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

	"gitlab.com/gitlab-org/cli/internal/run"
	"gitlab.com/gitlab-org/cli/internal/testing/cmdtest"

	"github.com/stretchr/testify/assert"
	"github.com/survivorbat/huhtest"
	"gitlab.com/gitlab-org/cli/internal/testing/httpmock"
	"gitlab.com/gitlab-org/cli/test"
)

type responseJSON struct {
	Ref string `json:"ref"`
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
		{
			name:             "when running `ci run` with variables",
			cli:              "-b main --variables FOO:bar",
			expectedPOSTBody: `"ref":"main","variables":[{"key":"FOO","value":"bar","variable_type":"env_var"}]`,
			expectedOut:      "Created pipeline (id: 123), status: created, ref: main, weburl: https://gitlab.com/OWNER/REPO/-/pipelines/123\n",
		},
		{
			name:             "when running `ci run` with multiple variables",
			cli:              "-b main --variables FOO:bar --variables BAR:xxx",
			expectedPOSTBody: `"ref":"main","variables":[{"key":"FOO","value":"bar","variable_type":"env_var"},{"key":"BAR","value":"xxx","variable_type":"env_var"}]`,
			expectedOut:      "Created pipeline (id: 123), status: created, ref: main, weburl: https://gitlab.com/OWNER/REPO/-/pipelines/123\n",
		},
		{
			name:             "when running `ci run` with variables-env",
			cli:              "-b main --variables-env FOO:bar",
			expectedPOSTBody: `"ref":"main","variables":[{"key":"FOO","value":"bar","variable_type":"env_var"}]`,
			expectedOut:      "Created pipeline (id: 123), status: created, ref: main, weburl: https://gitlab.com/OWNER/REPO/-/pipelines/123\n",
		},
		{
			name:             "when running `ci run` with multiple variables-env",
			cli:              "-b main --variables-env FOO:bar --variables-env BAR:xxx",
			expectedPOSTBody: `"ref":"main","variables":[{"key":"FOO","value":"bar","variable_type":"env_var"},{"key":"BAR","value":"xxx","variable_type":"env_var"}]`,
			expectedOut:      "Created pipeline (id: 123), status: created, ref: main, weburl: https://gitlab.com/OWNER/REPO/-/pipelines/123\n",
		},
		{
			name:             "when running `ci run` with mixed variables-env and variables",
			cli:              "-b main --variables-env FOO:bar --variables BAR:xxx",
			expectedPOSTBody: `"ref":"main","variables":[{"key":"FOO","value":"bar","variable_type":"env_var"},{"key":"BAR","value":"xxx","variable_type":"env_var"}]`,
			expectedOut:      "Created pipeline (id: 123), status: created, ref: main, weburl: https://gitlab.com/OWNER/REPO/-/pipelines/123\n",
		},
		{
			name:             "when running `ci run` with untyped input",
			cli:              "-b main -i key1:val1 --input key2:val2",
			expectedPOSTBody: `"ref":"main","inputs":{"key1":"val1","key2":"val2"}`,
			expectedOut:      "Created pipeline (id: 123), status: created, ref: main, weburl: https://gitlab.com/OWNER/REPO/-/pipelines/123\n",
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

					var response responseJSON
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

			execFunc := cmdtest.SetupCmdForTest(t, NewCmdRun, true,
				cmdtest.WithGitLabClient(cmdtest.NewTestApiClient(t, &http.Client{Transport: fakeHTTP}, "", glinstance.DefaultHostname).Lab()),
				cmdtest.WithBranch("custom-branch-123"),
			)
			restoreCmd := run.SetPrepareCmd(func(cmd *exec.Cmd) run.Runnable {
				return &test.OutputStub{}
			})
			t.Cleanup(restoreCmd)

			out, err := execFunc(tc.cli)

			assert.NoErrorf(t, err, "error running command `ci run %s`: %v", tc.cli, err)

			assert.Equal(t, tc.expectedOut, out.OutBuf.String())
			assert.Equal(t, tc.expectedErr, out.ErrBuf.String())
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
			expectedErr: "branch_without_mrs",
			mrIid:       1234,
		},
		{
			name:        "MR with variable flag",
			cli:         "--mr --variables key:val",
			expectedErr: "if any flags in the group [mr variables] are set none of the others can be",
			mrIid:       1235,
		},
		{
			name:        "MR with input flag",
			cli:         "--mr --input key:val",
			expectedErr: "if any flags in the group [mr input] are set none of the others can be",
			mrIid:       1236,
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
			execFunc := cmdtest.SetupCmdForTest(t, NewCmdRun, true,
				cmdtest.WithGitLabClient(cmdtest.NewTestApiClient(t, &http.Client{Transport: fakeHTTP}, "", glinstance.DefaultHostname).Lab()),
				cmdtest.WithBranch("custom-branch-123"),
				cmdtest.WithResponder(t, huhtest.NewResponder().AddSelect("Multiple merge requests exist for this branch", 0).MatchRegexp()),
			)
			restoreCmd := run.SetPrepareCmd(func(cmd *exec.Cmd) run.Runnable {
				return &test.OutputStub{}
			})
			t.Cleanup(restoreCmd)

			out, err := execFunc(tc.cli)

			if tc.expectedErr == "" {
				assert.NoErrorf(t, err, "error running command `ci run %s`: %v", tc.cli, err)

				assert.Contains(t, out.OutBuf.String(), tc.expectedOut)
				assert.Equal(t, tc.expectedErr, out.ErrBuf.String())
			} else {
				assert.ErrorContains(t, err, tc.expectedErr)
			}
		})
	}
}
