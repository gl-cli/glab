package update

import (
	"bytes"
	"net/http"
	"testing"

	"gitlab.com/gitlab-org/cli/commands/cmdtest"
	"gitlab.com/gitlab-org/cli/pkg/iostreams"

	"github.com/google/shlex"
	"github.com/stretchr/testify/assert"
	gitlab "gitlab.com/gitlab-org/api/client-go"
	"gitlab.com/gitlab-org/cli/api"
	"gitlab.com/gitlab-org/cli/internal/glrepo"
	"gitlab.com/gitlab-org/cli/pkg/httpmock"
)

func Test_NewCmdUpdate(t *testing.T) {
	tests := []struct {
		name     string
		cli      string
		wants    UpdateOpts
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
			wants: UpdateOpts{
				Key:       "cool_secret",
				Protected: true,
				Value:     "a secret",
				Group:     "",
				Type:      "env_var",
				Scope:     "*",
			},
		},
		{
			name: "protected var in group",
			cli:  `cool_secret --group coolGroup -v"cool"`,
			wants: UpdateOpts{
				Key:       "cool_secret",
				Protected: false,
				Value:     "cool",
				Group:     "coolGroup",
				Type:      "env_var",
				Scope:     "*",
			},
		},
		{
			name: "raw variable with flag",
			cli:  `cool_secret -r -v"$variable_name"`,
			wants: UpdateOpts{
				Key:   "cool_secret",
				Value: "$variable_name",
				Raw:   true,
				Group: "",
				Scope: "*",
				Type:  "env_var",
			},
		},
		{
			name: "raw variable with flag in group",
			cli:  `cool_secret -r --group coolGroup -v"$variable_name"`,
			wants: UpdateOpts{
				Key:   "cool_secret",
				Value: "$variable_name",
				Raw:   true,
				Group: "coolGroup",
				Scope: "*",
				Type:  "env_var",
			},
		},
		{
			name: "raw is false by default",
			cli:  `cool_secret -v"$variable_name"`,
			wants: UpdateOpts{
				Key:   "cool_secret",
				Value: "$variable_name",
				Raw:   false,
				Group: "",
				Scope: "*",
				Type:  "env_var",
			},
		},
		{
			name: "var with desription",
			cli:  `cool_secret -d"description"`,
			wants: UpdateOpts{
				Key:         "cool_secret",
				Raw:         false,
				Group:       "",
				Scope:       "*",
				Type:        "env_var",
				Description: "description",
			},
		},
		{
			name: "leading numbers in name",
			cli:  `123_TOKEN -v"cool"`,
			wants: UpdateOpts{
				Key:       "123_TOKEN",
				Protected: false,
				Value:     "cool",
				Group:     "",
				Scope:     "*",
				Type:      "env_var",
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
			io, _, _, _ := iostreams.Test()
			f := &cmdtest.Factory{
				IOStub: io,
			}

			io.IsInTTY = tt.stdinTTY

			argv, err := shlex.Split(tt.cli)
			assert.NoError(t, err)

			var gotOpts *UpdateOpts
			cmd := NewCmdUpdate(f, func(opts *UpdateOpts) error {
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

			assert.Equal(t, tt.wants.Key, gotOpts.Key)
			assert.Equal(t, tt.wants.Value, gotOpts.Value)
			assert.Equal(t, tt.wants.Group, gotOpts.Group)
			assert.Equal(t, tt.wants.Protected, gotOpts.Protected)
			assert.Equal(t, tt.wants.Raw, gotOpts.Raw)
			assert.Equal(t, tt.wants.Masked, gotOpts.Masked)
			assert.Equal(t, tt.wants.Type, gotOpts.Type)
			assert.Equal(t, tt.wants.Description, gotOpts.Description)
			assert.Equal(t, tt.wants.Scope, gotOpts.Scope)
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

	io, _, stdout, _ := iostreams.Test()

	opts := &UpdateOpts{
		HTTPClient: func() (*gitlab.Client, error) {
			a, _ := api.TestClient(&http.Client{Transport: reg}, "", "gitlab.com", false)
			return a.Lab(), nil
		},
		BaseRepo: func() (glrepo.Interface, error) {
			return glrepo.FromFullName("owner/repo")
		},
		IO:    io,
		Key:   "TEST_VARIABLE",
		Value: "foo",
		Scope: "*",
	}
	_, _ = opts.HTTPClient()

	err := updateRun(opts)
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

	io, _, stdout, _ := iostreams.Test()

	opts := &UpdateOpts{
		HTTPClient: func() (*gitlab.Client, error) {
			a, _ := api.TestClient(&http.Client{Transport: reg}, "", "gitlab.com", false)
			return a.Lab(), nil
		},
		BaseRepo: func() (glrepo.Interface, error) {
			return glrepo.FromFullName("owner/repo")
		},
		IO:    io,
		Key:   "TEST_VARIABLE",
		Value: "blargh",
		Group: "mygroup",
	}
	_, _ = opts.HTTPClient()

	err := updateRun(opts)
	assert.NoError(t, err)
	assert.Equal(t, stdout.String(), "✓ Updated variable TEST_VARIABLE for group mygroup.\n")
}
