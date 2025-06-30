package delete

import (
	"bytes"
	"net/http"
	"testing"

	gitlabtesting "gitlab.com/gitlab-org/api/client-go/testing"
	"gitlab.com/gitlab-org/cli/api"
	"gitlab.com/gitlab-org/cli/internal/commands/cmdtest"
	"gitlab.com/gitlab-org/cli/internal/config"
	"gitlab.com/gitlab-org/cli/internal/glrepo"
	"go.uber.org/mock/gomock"

	"github.com/google/shlex"
	"github.com/stretchr/testify/assert"
	"gitlab.com/gitlab-org/cli/internal/glinstance"
	"gitlab.com/gitlab-org/cli/internal/httpmock"
)

func Test_NewCmdDelete(t *testing.T) {
	tests := []struct {
		name     string
		cli      string
		wants    options
		stdinTTY bool
		wantsErr bool
	}{
		{
			name:     "delete var",
			cli:      "cool_secret",
			wantsErr: false,
		},
		{
			name:     "delete scoped var",
			cli:      "cool_secret --scope prod",
			wantsErr: false,
		},
		{
			name:     "delete group var",
			cli:      "cool_secret -g mygroup",
			wantsErr: false,
		},
		{
			name:     "delete scoped group var",
			cli:      "cool_secret -g mygroup --scope prod",
			wantsErr: true,
		},
		{
			name:     "no name",
			cli:      "",
			wantsErr: true,
		},
		{
			name:     "invalid characters in name",
			cli:      "BAD-SECRET",
			wantsErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			io, _, _, _ := cmdtest.TestIOStreams()
			f := &cmdtest.Factory{
				IOStub: io,
				ApiClientStub: func(repoHost string, cfg config.Config) (*api.Client, error) {
					tc := gitlabtesting.NewTestClient(t)
					tc.MockProjectVariables.EXPECT().RemoveVariable(gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes()
					tc.MockGroupVariables.EXPECT().RemoveVariable(gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes()
					return cmdtest.TestClient(&http.Client{}, "", repoHost, false, api.WithGitLabClient(tc.Client))
				},
				BaseRepoStub: func() (glrepo.Interface, error) {
					return glrepo.New("OWNER", "REPO", glinstance.DefaultHostname), nil
				},
				ConfigStub: func() config.Config {
					return config.NewBlankConfig()
				},
			}

			io.IsInTTY = tt.stdinTTY

			argv, err := shlex.Split(tt.cli)
			assert.NoError(t, err)

			cmd := NewCmdDelete(f)
			cmd.SetArgs(argv)
			cmd.SetIn(&bytes.Buffer{})
			cmd.SetOut(&bytes.Buffer{})
			cmd.SetErr(&bytes.Buffer{})

			_, err = cmd.ExecuteC()
			if tt.wantsErr {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)
		})
	}
}

func Test_deleteRun(t *testing.T) {
	reg := &httpmock.Mocker{
		MatchURL: httpmock.PathAndQuerystring,
	}
	defer reg.Verify(t)

	reg.RegisterResponder(http.MethodDelete, "/api/v4/projects/owner%2Frepo/variables/TEST_VAR?filter%5Benvironment_scope%5D=%2A",
		httpmock.NewStringResponse(http.StatusNoContent, " "),
	)

	reg.RegisterResponder(http.MethodDelete, "/api/v4/projects/owner%2Frepo/variables/TEST_VAR?filter%5Benvironment_scope%5D=stage",
		httpmock.NewStringResponse(http.StatusNoContent, " "),
	)

	reg.RegisterResponder(http.MethodDelete, "/api/v4/groups/testGroup/variables/TEST_VAR",
		httpmock.NewStringResponse(http.StatusNoContent, " "),
	)

	apiClient := func(repoHost string, cfg config.Config) (*api.Client, error) {
		a, _ := cmdtest.TestClient(&http.Client{Transport: reg}, "", "gitlab.com", false)
		return a, nil
	}
	baseRepo := func() (glrepo.Interface, error) {
		return glrepo.FromFullName("owner/repo", glinstance.DefaultHostname)
	}

	tests := []struct {
		name        string
		opts        options
		wantsErr    bool
		wantsOutput string
	}{
		{
			name: "delete project variable no scope",
			opts: options{
				apiClient: apiClient,
				config:    config.NewBlankConfig(),
				baseRepo:  baseRepo,
				key:       "TEST_VAR",
				scope:     "*",
			},
			wantsErr:    false,
			wantsOutput: "✓ Deleted variable TEST_VAR with scope * for owner/repo.\n",
		},
		{
			name: "delete project variable with stage scope",
			opts: options{
				apiClient: apiClient,
				config:    config.NewBlankConfig(),
				baseRepo:  baseRepo,
				key:       "TEST_VAR",
				scope:     "stage",
			},
			wantsErr:    false,
			wantsOutput: "✓ Deleted variable TEST_VAR with scope stage for owner/repo.\n",
		},
		{
			name: "delete group variable",
			opts: options{
				apiClient: apiClient,
				config:    config.NewBlankConfig(),
				baseRepo:  baseRepo,
				key:       "TEST_VAR",
				scope:     "",
				group:     "testGroup",
			},
			wantsErr:    false,
			wantsOutput: "✓ Deleted variable TEST_VAR for group testGroup.\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			io, _, stdout, _ := cmdtest.TestIOStreams()
			tt.opts.io = io

			err := tt.opts.run()
			assert.NoError(t, err)
			assert.Equal(t, stdout.String(), tt.wantsOutput)
		})
	}
}
