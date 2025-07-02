package update

import (
	"bytes"
	"net/http"
	"testing"

	"gitlab.com/gitlab-org/cli/internal/api"
	"gitlab.com/gitlab-org/cli/internal/testing/cmdtest"

	"github.com/google/shlex"
	"github.com/stretchr/testify/assert"
	"gitlab.com/gitlab-org/cli/internal/config"
	"gitlab.com/gitlab-org/cli/internal/glinstance"
	"gitlab.com/gitlab-org/cli/internal/glrepo"
	"gitlab.com/gitlab-org/cli/internal/testing/httpmock"
)

func Test_NewCmdUpdate(t *testing.T) {
	tests := []struct {
		name     string
		cli      string
		wants    options
		stdinTTY bool
		wantsErr bool
	}{
		{
			name:     "invalid type",
			cli:      "cool_secret -g coolGroup -t 'mV'",
			wantsErr: true,
		},
		{
			name:     "value argument and value flag specified",
			cli:      `cool_secret value -v"another value"`,
			wantsErr: true,
		},
		{
			name:     "no name",
			cli:      "",
			wantsErr: true,
		},
		{
			name:     "no value, stdin is terminal",
			cli:      "cool_secret",
			stdinTTY: true,
			wantsErr: true,
		},
		{
			name: "protected var",
			cli:  `cool_secret -v"a secret" -p`,
			wants: options{
				key:       "cool_secret",
				protected: true,
				value:     "a secret",
				group:     "",
				typ:       "env_var",
				scope:     "*",
			},
		},
		{
			name: "protected var in group",
			cli:  `cool_secret --group coolGroup -v"cool"`,
			wants: options{
				key:       "cool_secret",
				protected: false,
				value:     "cool",
				group:     "coolGroup",
				typ:       "env_var",
				scope:     "*",
			},
		},
		{
			name: "raw variable with flag",
			cli:  `cool_secret -r -v"$variable_name"`,
			wants: options{
				key:   "cool_secret",
				value: "$variable_name",
				raw:   true,
				group: "",
				scope: "*",
				typ:   "env_var",
			},
		},
		{
			name: "raw variable with flag in group",
			cli:  `cool_secret -r --group coolGroup -v"$variable_name"`,
			wants: options{
				key:   "cool_secret",
				value: "$variable_name",
				raw:   true,
				group: "coolGroup",
				scope: "*",
				typ:   "env_var",
			},
		},
		{
			name: "raw is false by default",
			cli:  `cool_secret -v"$variable_name"`,
			wants: options{
				key:   "cool_secret",
				value: "$variable_name",
				raw:   false,
				group: "",
				scope: "*",
				typ:   "env_var",
			},
		},
		{
			name: "var with desription",
			cli:  `cool_secret -d"description"`,
			wants: options{
				key:         "cool_secret",
				raw:         false,
				group:       "",
				scope:       "*",
				typ:         "env_var",
				description: "description",
			},
		},
		{
			name: "leading numbers in name",
			cli:  `123_TOKEN -v"cool"`,
			wants: options{
				key:       "123_TOKEN",
				protected: false,
				value:     "cool",
				group:     "",
				scope:     "*",
				typ:       "env_var",
			},
		},
		{
			name:     "invalid characters in name",
			cli:      `BAD-SECRET -v"cool"`,
			wantsErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			io, _, _, _ := cmdtest.TestIOStreams()
			f := cmdtest.NewTestFactory(io)

			io.IsInTTY = tt.stdinTTY

			argv, err := shlex.Split(tt.cli)
			assert.NoError(t, err)

			var gotOpts *options
			cmd := NewCmdUpdate(f, func(opts *options) error {
				gotOpts = opts
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

			assert.Equal(t, tt.wants.key, gotOpts.key)
			assert.Equal(t, tt.wants.value, gotOpts.value)
			assert.Equal(t, tt.wants.group, gotOpts.group)
			assert.Equal(t, tt.wants.protected, gotOpts.protected)
			assert.Equal(t, tt.wants.raw, gotOpts.raw)
			assert.Equal(t, tt.wants.masked, gotOpts.masked)
			assert.Equal(t, tt.wants.typ, gotOpts.typ)
			assert.Equal(t, tt.wants.description, gotOpts.description)
			assert.Equal(t, tt.wants.scope, gotOpts.scope)
		})
	}
}

func Test_updateRun_project(t *testing.T) {
	reg := &httpmock.Mocker{}
	defer reg.Verify(t)

	reg.RegisterResponder(http.MethodPut, "/projects/owner/repo/variables/TEST_VARIABLE",
		httpmock.NewStringResponse(http.StatusCreated, `
			{
    			"key": "TEST_VARIABLE",
    			"value": "foo",
    			"variable_type": "env_var",
    			"protected": false,
    			"masked": false
			}
		`),
	)

	io, _, stdout, _ := cmdtest.TestIOStreams()

	opts := &options{
		apiClient: func(repoHost string, cfg config.Config) (*api.Client, error) {
			return cmdtest.NewTestApiClient(t, &http.Client{Transport: reg}, "", "gitlab.com"), nil
		},
		baseRepo: func() (glrepo.Interface, error) {
			return glrepo.FromFullName("owner/repo", glinstance.DefaultHostname)
		},
		config: config.NewBlankConfig(),
		io:     io,
		key:    "TEST_VARIABLE",
		value:  "foo",
		scope:  "*",
	}

	err := opts.run()
	assert.NoError(t, err)
	assert.Equal(t, stdout.String(), "✓ Updated variable TEST_VARIABLE for project owner/repo with scope *.\n")
}

func Test_updateRun_group(t *testing.T) {
	reg := &httpmock.Mocker{}
	defer reg.Verify(t)

	reg.RegisterResponder(http.MethodPut, "/groups/mygroup/variables/TEST_VARIABLE",
		httpmock.NewStringResponse(http.StatusCreated, `
			{
    			"key": "TEST_VARIABLE",
    			"value": "blargh",
    			"variable_type": "env_var",
    			"protected": false,
    			"masked": false
			}
		`),
	)

	io, _, stdout, _ := cmdtest.TestIOStreams()

	opts := &options{
		apiClient: func(repoHost string, cfg config.Config) (*api.Client, error) {
			return cmdtest.NewTestApiClient(t, &http.Client{Transport: reg}, "", "gitlab.com"), nil
		},
		baseRepo: func() (glrepo.Interface, error) {
			return glrepo.FromFullName("owner/repo", glinstance.DefaultHostname)
		},
		config: config.NewBlankConfig(),
		io:     io,
		key:    "TEST_VARIABLE",
		value:  "blargh",
		group:  "mygroup",
	}

	err := opts.run()
	assert.NoError(t, err)
	assert.Equal(t, stdout.String(), "✓ Updated variable TEST_VARIABLE for group mygroup.\n")
}
