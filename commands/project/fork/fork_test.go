package fork

import (
	"net/http"
	"strings"
	"testing"

	"gitlab.com/gitlab-org/cli/commands/cmdtest"
	"gitlab.com/gitlab-org/cli/pkg/httpmock"
	"gitlab.com/gitlab-org/cli/pkg/prompt"
	"gitlab.com/gitlab-org/cli/test"

	"github.com/stretchr/testify/assert"
)

func runCommand(rt http.RoundTripper, isTTY bool, cli string) (*test.CmdOut, error) {
	ios, _, stdout, stderr := cmdtest.InitIOStreams(isTTY, "")

	factory := cmdtest.InitFactory(ios, rt)

	_, _ = factory.HttpClient()

	cmd := NewCmdFork(factory, nil)

	return cmdtest.ExecuteCommand(cmd, cli, stdout, stderr)
}

func TestProjectFork(t *testing.T) {
	type httpMock struct {
		method string
		path   string
		status int
		body   string
	}

	httpMocks := []httpMock{
		{
			http.MethodGet,
			"/api/v4/user",
			http.StatusOK,
			`{ "username": "OWNER" }`,
		},
		{
			http.MethodPost,
			"/api/v4/projects/OWNER/REPO/fork",
			http.StatusCreated,
			`{"id": 99}`,
		},
		{
			http.MethodGet,
			"/api/v4/projects/99?license=true&with_custom_attributes=true",
			http.StatusOK,
			`{
							"id": 99,
							"import_status": "finished",
							"ssh_url_to_repo": "git@gitlab.com:OWNER/REPO.git",
							"path_with_namespace": "OWNER/baz"
							}`,
		},
	}

	cloneShelloutStubs := []string{
		"git clone executed",
		"git remote added",
	}

	expectedCloneShellouts := []string{
		"git clone git@gitlab.com:OWNER/REPO.git",
		"git -C REPO remote add -f upstream git@gitlab.com:OWNER/baz.git",
	}

	tests := []struct {
		name              string
		commandArgs       string
		shelloutStubs     []string
		expectedShellouts []string
		expectClonePrompt bool
		expectClone       bool
	}{
		{
			name:              "when a specified repository is forked and cloned",
			commandArgs:       "OWNER/REPO --name foo --path baz --clone",
			shelloutStubs:     cloneShelloutStubs,
			expectedShellouts: expectedCloneShellouts,
			expectClonePrompt: false,
			expectClone:       true,
		},
		{
			name:              "when a specified repository is forked user is prompted to clone",
			commandArgs:       "OWNER/REPO --name foo --path baz",
			shelloutStubs:     cloneShelloutStubs,
			expectedShellouts: expectedCloneShellouts,
			expectClonePrompt: true,
			expectClone:       true,
		},
		{
			name:              "when a specified repository is forked and clone is set to false",
			commandArgs:       "OWNER/REPO --name foo --path baz --clone=false",
			shelloutStubs:     []string{},
			expectedShellouts: []string{},
			expectClonePrompt: false,
			expectClone:       false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			fakeHTTP := &httpmock.Mocker{
				MatchURL: httpmock.PathAndQuerystring,
			}
			defer fakeHTTP.Verify(t)

			for _, mock := range httpMocks {
				fakeHTTP.RegisterResponder(mock.method, mock.path, httpmock.NewStringResponse(mock.status, mock.body))
			}

			if tc.expectClonePrompt {
				restore := prompt.StubConfirm(true)
				defer restore()
			}

			if tc.expectClone {
				fakeHTTP.RegisterResponder(http.MethodGet, "/api/v4/projects/OWNER/REPO?license=true&with_custom_attributes=true",
					httpmock.NewStringResponse(http.StatusOK, `{
							  "id": 100,
							  "description": "this is a test description",
							  "name": "foo",
							  "name_with_namespace": "OWNER / baz",
							  "path": "baz",
							  "path_with_namespace": "OWNER/baz",
							  "created_at": "2022-07-13T02:04:56.151Z",
							  "default_branch": "main",
							  "http_url_to_repo": "https://gitlab.com/OWNER/baz.git",
							  "ssh_url_to_repo": "git@gitlab.com:OWNER/baz.git"
							}`))
			}

			cs, csTeardown := test.InitCmdStubber()
			defer csTeardown()
			for _, stub := range tc.shelloutStubs {
				cs.Stub(stub)
			}

			output, err := runCommand(fakeHTTP, false, tc.commandArgs)

			if assert.NoErrorf(t, err, "error running command `project fork %s`: %v", tc.commandArgs, err) {
				assert.Empty(t, output.String())
				assert.Equal(t, "- finished\n✓ Created fork OWNER/baz.\n", output.Stderr())
			}

			assert.Equal(t, len(tc.expectedShellouts), cs.Count)
			for idx, expectedShellout := range tc.expectedShellouts {
				assert.Equal(t, expectedShellout, strings.Join(cs.Calls[idx].Args, " "))
			}
		})
	}
}
