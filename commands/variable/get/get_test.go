package get

import (
	"bytes"
	"net/http"
	"testing"

	"github.com/google/shlex"
	"github.com/stretchr/testify/assert"
	gitlab "gitlab.com/gitlab-org/api/client-go"
	"gitlab.com/gitlab-org/cli/commands/cmdtest"
	"gitlab.com/gitlab-org/cli/internal/glrepo"
	"gitlab.com/gitlab-org/cli/pkg/httpmock"
	"gitlab.com/gitlab-org/cli/pkg/iostreams"
)

func Test_NewCmdGet(t *testing.T) {
	tests := []struct {
		name     string
		cli      string
		wants    options
		wantsErr bool
	}{
		{
			name:     "good key",
			cli:      "good_key",
			wantsErr: false,
			wants: options{
				key: "good_key",
			},
		},
		{
			name:     "bad key",
			cli:      "bad-key",
			wantsErr: true,
		},
		{
			name:     "no key",
			cli:      "",
			wantsErr: true,
		},
		{
			name: "good key",
			cli:  "-g group good_key",
			wants: options{
				key:   "good_key",
				group: "group",
			},
			wantsErr: false,
		},
		{
			name: "good key, with scope",
			cli:  "-s foo -g group good_key",
			wants: options{
				key:   "good_key",
				group: "group",
				scope: "foo",
			},
			wantsErr: false,
		},
		{
			name: "good key, with default scope",
			cli:  "-g group good_key",
			wants: options{
				key:   "good_key",
				group: "group",
				scope: "*",
			},
			wantsErr: false,
		},
		{
			name: "bad key",
			cli:  "-g group bad-key",
			wants: options{
				group: "group",
			},
			wantsErr: true,
		},
		{
			name:     "good key but no group",
			cli:      "good_key --group",
			wantsErr: true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			io, _, _, _ := iostreams.Test()
			f := &cmdtest.Factory{
				IOStub: io,
			}

			argv, err := shlex.Split(test.cli)
			assert.NoError(t, err)

			var gotOpts *options
			cmd := NewCmdGet(f, func(opts *options) error {
				gotOpts = opts
				return nil
			})
			cmd.SetArgs(argv)
			cmd.SetIn(&bytes.Buffer{})
			cmd.SetOut(&bytes.Buffer{})
			cmd.SetErr(&bytes.Buffer{})

			_, err = cmd.ExecuteC()
			if test.wantsErr {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)

			assert.Equal(t, test.wants.key, gotOpts.key)
			assert.Equal(t, test.wants.group, gotOpts.group)
		})
	}
}

func Test_getRun_project(t *testing.T) {
	reg := &httpmock.Mocker{}
	defer reg.Verify(t)

	varContent := `
			TEST variable\n
			content
		`

	body := struct {
		Key              string `json:"key"`
		VariableType     string `json:"variable_type"`
		Value            string `json:"value"`
		Protected        bool   `json:"protected"`
		Masked           bool   `json:"masked"`
		EnvironmentScope string `json:"environment_scope"`
	}{
		Key:              "TEST_VAR",
		VariableType:     "env_var",
		Value:            varContent,
		Protected:        false,
		Masked:           false,
		EnvironmentScope: "*",
	}

	reg.RegisterResponder(http.MethodGet, "/projects/owner/repo/variables/TEST_VAR",
		httpmock.NewJSONResponse(http.StatusOK, body),
	)

	io, _, stdout, _ := iostreams.Test()

	opts := &options{
		httpClient: func() (*gitlab.Client, error) {
			a, _ := cmdtest.TestClient(&http.Client{Transport: reg}, "", "gitlab.com", false)
			return a.Lab(), nil
		},
		baseRepo: func() (glrepo.Interface, error) {
			return glrepo.FromFullName("owner/repo")
		},
		io:  io,
		key: "TEST_VAR",
	}
	_, _ = opts.httpClient()

	err := opts.run()
	assert.NoError(t, err)
	assert.Equal(t, varContent, stdout.String())
}
