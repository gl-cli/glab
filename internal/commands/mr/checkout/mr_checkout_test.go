//go:build !integration

package checkout

import (
	"net/http"
	"net/url"
	"strings"
	"testing"

	"github.com/MakeNowJust/heredoc/v2"
	"github.com/stretchr/testify/assert"

	"gitlab.com/gitlab-org/cli/internal/config"
	"gitlab.com/gitlab-org/cli/internal/git"
	"gitlab.com/gitlab-org/cli/internal/glinstance"
	"gitlab.com/gitlab-org/cli/internal/glrepo"
	"gitlab.com/gitlab-org/cli/internal/testing/cmdtest"
	"gitlab.com/gitlab-org/cli/internal/testing/httpmock"
	"gitlab.com/gitlab-org/cli/test"
)

func runCommand(t *testing.T, rt http.RoundTripper, branch string, cli string, opts ...cmdtest.FactoryOption) (*test.CmdOut, error) {
	t.Helper()

	// Default options
	defaultOpts := []cmdtest.FactoryOption{
		cmdtest.WithGitLabClient(cmdtest.NewTestApiClient(t, &http.Client{Transport: rt}, "", glinstance.DefaultHostname).Lab()),
		cmdtest.WithBranch(branch),
		func(f *cmdtest.Factory) {
			f.RemotesStub = func() (glrepo.Remotes, error) {
				pu, _ := url.Parse("https://gitlab.com/OWNER/REPO.git")

				return glrepo.Remotes{
					{
						Remote: &git.Remote{
							Name:     "upstream",
							Resolved: "base",
							PushURL:  pu,
						},
						Repo: glrepo.New("OWNER", "REPO", glinstance.DefaultHostname),
					},
					{
						Remote: &git.Remote{
							Name:     "origin",
							Resolved: "base",
							PushURL:  pu,
						},
						Repo: glrepo.New("monalisa", "REPO", glinstance.DefaultHostname),
					},
				}, nil
			}
		},
	}

	exec := cmdtest.SetupCmdForTest(t, NewCmdCheckout, false, append(defaultOpts, opts...)...)
	return exec(cli)
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

			output, err := runCommand(t, fakeHTTP, tc.branch, tc.commandArgs)

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

func TestMrCheckout_HTTPSProtocolConfiguration(t *testing.T) {
	fakeHTTP := httpmock.New()
	defer fakeHTTP.Verify(t)

	fakeHTTP.RegisterResponder(
		http.MethodGet,
		"/api/v4/projects/OWNER/REPO/merge_requests/123",
		httpmock.NewStringResponse(http.StatusOK, `{
			"id": 123,
			"iid": 123,
			"project_id": 3,
			"source_project_id": 3,
			"title": "test mr title",
			"description": "test mr description",
			"allow_collaboration": false,
			"state": "opened",
			"source_branch":"feat-new-mr"
		}`),
	)

	fakeHTTP.RegisterResponder(
		http.MethodGet,
		"/api/v4/projects/3",
		httpmock.NewStringResponse(http.StatusOK, `{
			"id": 3,
			"http_url_to_repo": "https://gitlab.com/OWNER/REPO.git",
			"ssh_url_to_repo": "git@gitlab.com:OWNER/REPO.git"
		}`),
	)

	cs, csTeardown := test.InitCmdStubber()
	defer csTeardown()

	cs.Stub("HEAD branch: master\n")
	cs.Stub("\n")
	cs.Stub("\n")
	cs.Stub(heredoc.Doc(`
		deadbeef HEAD
		deadb00f refs/remotes/upstream/feat-new-mr
		deadbeef refs/remotes/origin/feat-new-mr
	`))

	// Create config with HTTPS protocol
	cfg := config.NewBlankConfig()
	err := cfg.Set("gitlab.com", "git_protocol", "https")
	assert.NoError(t, err)

	output, err := runCommand(t, fakeHTTP, "main", "123", cmdtest.WithConfig(cfg))

	assert.NoError(t, err)
	assert.Empty(t, output.String())
	assert.Empty(t, output.Stderr())

	expectedShellouts := []string{
		"git fetch https://gitlab.com/OWNER/REPO.git refs/heads/feat-new-mr:feat-new-mr",
		"git config branch.feat-new-mr.remote https://gitlab.com/OWNER/REPO.git",
		"git config branch.feat-new-mr.merge refs/heads/feat-new-mr",
		"git checkout feat-new-mr",
	}

	assert.Equal(t, len(expectedShellouts), cs.Count)
	for idx, expectedShellout := range expectedShellouts {
		assert.Equal(t, expectedShellout, strings.Join(cs.Calls[idx].Args, " "))
	}
}
