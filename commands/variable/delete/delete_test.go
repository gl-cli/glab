package delete

import (
	"bytes"
	"net/http"
	"testing"

	"github.com/xanzy/go-gitlab"
	"gitlab.com/gitlab-org/cli/api"
	"gitlab.com/gitlab-org/cli/internal/glrepo"
	"gitlab.com/gitlab-org/cli/pkg/iostreams"

	"github.com/google/shlex"
	"github.com/stretchr/testify/assert"
	"gitlab.com/gitlab-org/cli/commands/cmdutils"
	"gitlab.com/gitlab-org/cli/pkg/httpmock"
)

func Test_NewCmdSet(t *testing.T) {
	tests := []struct {
		name     string
		cli      string
		wants    DeleteOpts
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
			io, _, _, _ := iostreams.Test()
			f := &cmdutils.Factory{
				IO: io,
			}

			io.IsInTTY = tt.stdinTTY

			argv, err := shlex.Split(tt.cli)
			assert.NoError(t, err)

			cmd := NewCmdSet(f, func(opts *DeleteOpts) error {
				return nil
			})

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

	httpClient := func() (*gitlab.Client, error) {
		a, _ := api.TestClient(&http.Client{Transport: reg}, "", "gitlab.com", false)
		return a.Lab(), nil
	}
	baseRepo := func() (glrepo.Interface, error) {
		return glrepo.FromFullName("owner/repo")
	}

	tests := []struct {
		name        string
		opts        DeleteOpts
		wantsErr    bool
		wantsOutput string
	}{
		{
			name: "delete project variable no scope",
			opts: DeleteOpts{
				HTTPClient: httpClient,
				BaseRepo:   baseRepo,
				Key:        "TEST_VAR",
				Scope:      "*",
			},
			wantsErr:    false,
			wantsOutput: "✓ Deleted variable TEST_VAR with scope * for owner/repo.\n",
		},
		{
			name: "delete project variable with stage scope",
			opts: DeleteOpts{
				HTTPClient: httpClient,
				BaseRepo:   baseRepo,
				Key:        "TEST_VAR",
				Scope:      "stage",
			},
			wantsErr:    false,
			wantsOutput: "✓ Deleted variable TEST_VAR with scope stage for owner/repo.\n",
		},
		{
			name: "delete group variable",
			opts: DeleteOpts{
				HTTPClient: httpClient,
				BaseRepo:   baseRepo,
				Key:        "TEST_VAR",
				Scope:      "",
				Group:      "testGroup",
			},
			wantsErr:    false,
			wantsOutput: "✓ Deleted variable TEST_VAR for group testGroup.\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, _ = tt.opts.HTTPClient()

			io, _, stdout, _ := iostreams.Test()
			tt.opts.IO = io
			io.IsInTTY = false

			err := deleteRun(&tt.opts)
			assert.NoError(t, err)
			assert.Equal(t, stdout.String(), tt.wantsOutput)
		})
	}
}
