package checkout

import (
	"net/http"
	"net/url"
	"strings"
	"testing"

	"gitlab.com/gitlab-org/cli/commands/cmdtest"

	"github.com/MakeNowJust/heredoc/v2"
	"gitlab.com/gitlab-org/cli/pkg/git"

	"github.com/stretchr/testify/assert"
	"gitlab.com/gitlab-org/cli/internal/glrepo"
	"gitlab.com/gitlab-org/cli/pkg/httpmock"
	"gitlab.com/gitlab-org/cli/test"
)

func runCommand(rt http.RoundTripper, branch string, isTTY bool, cli string) (*test.CmdOut, error) {
	ios, _, stdout, stderr := cmdtest.InitIOStreams(isTTY, "")
	pu, _ := url.Parse("https://gitlab.com/OWNER/REPO.git")

	factory := cmdtest.InitFactory(ios, rt)

	factory.Remotes = func() (glrepo.Remotes, error) {
		return glrepo.Remotes{
			{
				Remote: &git.Remote{
					Name:     "upstream",
					Resolved: "base",
					PushURL:  pu,
				},
				Repo: glrepo.New("OWNER", "REPO"),
			},
			{
				Remote: &git.Remote{
					Name:     "origin",
					Resolved: "base",
					PushURL:  pu,
				},
				Repo: glrepo.New("monalisa", "REPO"),
			},
		}, nil
	}

	factory.Branch = func() (string, error) {
		return branch, nil
	}

	_, _ = factory.HttpClient()

	cmd := NewCmdCheckout(factory)

	return cmdtest.ExecuteCommand(cmd, cli, stdout, stderr)
}

func TestMrCheckout(t *testing.T) {
	type httpMock struct {
		method string
		path   string
		status int
		body   string
	}

	tests := []struct {
		name        string
		commandArgs string
		branch      string
		httpMocks   []httpMock

		shelloutStubs []string

		expectedShellouts []string
	}{
		{
			name:        "when a valid MR is checked out using MR id",
			commandArgs: "123",
			branch:      "main",
			httpMocks: []httpMock{
				{
					http.MethodGet,
					"/api/v4/projects/OWNER/REPO/merge_requests/123",
					http.StatusOK,
					`{
							"id": 123,
							"iid": 123,
							"project_id": 3,
							"source_project_id": 3,
							"title": "test mr title",
							"description": "test mr description",
							"allow_collaboration": false,
							"state": "opened",
							"source_branch":"feat-new-mr"
							}`,
				},
				{
					http.MethodGet,
					"/api/v4/projects/3",
					http.StatusOK,
					`{
							"id": 3,
							"ssh_url_to_repo": "git@gitlab.com:OWNER/REPO.git"
							}`,
				},
			},
			shelloutStubs: []string{
				"HEAD branch: master\n",
				"\n",
				"\n",
				heredoc.Doc(`
				deadbeef HEAD
				deadb00f refs/remotes/upstream/feat-new-mr
				deadbeef refs/remotes/origin/feat-new-mr
				`),
			},

			expectedShellouts: []string{
				"git fetch git@gitlab.com:OWNER/REPO.git refs/heads/feat-new-mr:feat-new-mr",
				"git config branch.feat-new-mr.remote git@gitlab.com:OWNER/REPO.git",
				"git config branch.feat-new-mr.merge refs/heads/feat-new-mr",
				"git checkout feat-new-mr",
			},
		},
		{
			name:        "when a valid MR comes from a forked private project",
			commandArgs: "123",
			branch:      "main",
			httpMocks: []httpMock{
				{
					http.MethodGet,
					"/api/v4/projects/OWNER/REPO/merge_requests/123",
					http.StatusOK,
					`{
							"id": 123,
							"iid": 123,
							"project_id": 3,
							"source_project_id": 3,
							"target_project_id": 4,
							"title": "test mr title",
							"description": "test mr description",
							"allow_collaboration": false,
							"state": "opened",
							"source_branch":"feat-new-mr"
							}`,
				},
				{
					http.MethodGet,
					"/api/v4/projects/4",
					http.StatusOK,
					`{
							"id": 4,
							"ssh_url_to_repo": "git@gitlab.com:OWNER/REPO.git"
						}`,
				},
				{
					http.MethodGet,
					"/api/v4/projects/3",
					http.StatusNotFound,
					`{
						"message":"404 Project Not Found"
					}`,
				},
			},
			shelloutStubs: []string{
				"HEAD branch: master\n",
				"\n",
				"\n",
				heredoc.Doc(`
				deadbeef HEAD
				deadb00f refs/remotes/upstream/feat-new-mr
				deadbeef refs/remotes/origin/feat-new-mr
				`),
			},

			expectedShellouts: []string{
				"git fetch git@gitlab.com:OWNER/REPO.git refs/merge-requests/123/head:feat-new-mr",
				"git config branch.feat-new-mr.remote git@gitlab.com:OWNER/REPO.git",
				"git config branch.feat-new-mr.merge refs/merge-requests/123/head",
				"git checkout feat-new-mr",
			},
		},
		{
			name:        "when a valid MR is checked out using MR id and specifying branch",
			commandArgs: "123 --branch foo",
			branch:      "main",
			httpMocks: []httpMock{
				{
					http.MethodGet,
					"/api/v4/projects/OWNER/REPO/merge_requests/123",
					http.StatusOK,
					`{
							"id": 123,
							"iid": 123,
							"project_id": 3,
							"source_project_id": 4,
							"title": "test mr title",
							"description": "test mr description",
							"allow_collaboration": true,
							"state": "opened",
							"source_branch":"feat-new-mr"
							}`,
				},
				{
					http.MethodGet,
					"/api/v4/projects/4",
					http.StatusOK,
					`{
							"id": 3,
							"ssh_url_to_repo": "git@gitlab.com:FORK_OWNER/REPO.git"
							}`,
				},
			},
			shelloutStubs: []string{
				"HEAD branch: master\n",
				"\n",
				"\n",
				"\n",
				heredoc.Doc(`
				deadbeef HEAD
				deadb00f refs/remotes/upstream/feat-new-mr
				deadbeef refs/remotes/origin/feat-new-mr
				`),
			},

			expectedShellouts: []string{
				"git fetch git@gitlab.com:FORK_OWNER/REPO.git refs/heads/feat-new-mr:foo",
				"git config branch.foo.remote git@gitlab.com:FORK_OWNER/REPO.git",
				"git config branch.foo.pushRemote git@gitlab.com:FORK_OWNER/REPO.git",
				"git config branch.foo.merge refs/heads/feat-new-mr",
				"git checkout foo",
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			fakeHTTP := httpmock.New()
			defer fakeHTTP.Verify(t)

			for _, mock := range tc.httpMocks {
				fakeHTTP.RegisterResponder(mock.method, mock.path, httpmock.NewStringResponse(mock.status, mock.body))
			}

			cs, csTeardown := test.InitCmdStubber()
			defer csTeardown()
			for _, stub := range tc.shelloutStubs {
				cs.Stub(stub)
			}

			output, err := runCommand(fakeHTTP, tc.branch, false, tc.commandArgs)

			if assert.NoErrorf(t, err, "error running command `mr checkout %s`: %v", tc.commandArgs, err) {
				assert.Empty(t, output.String())
				assert.Empty(t, output.Stderr())
			}

			assert.Equal(t, len(tc.expectedShellouts), cs.Count)
			for idx, expectedShellout := range tc.expectedShellouts {
				assert.Equal(t, expectedShellout, strings.Join(cs.Calls[idx].Args, " "))
			}
		})
	}
}
