package set

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

func Test_NewCmdSet(t *testing.T) {
	tests := []struct {
		name     string
		cli      string
		wants    SetOpts
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
			wants: SetOpts{
				Key:       "cool_secret",
				Protected: true,
				Value:     "a secret",
				Group:     "",
				Scope:     "*",
				Type:      "env_var",
			},
		},
		{
			name: "protected var in group",
			cli:  `cool_secret --group coolGroup -v"cool"`,
			wants: SetOpts{
				Key:       "cool_secret",
				Protected: false,
				Value:     "cool",
				Group:     "coolGroup",
				Scope:     "*",
				Type:      "env_var",
			},
		},
		{
			name: "leading numbers in name",
			cli:  `123_TOKEN -v"cool"`,
			wants: SetOpts{
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
		{
			name: "environment scope in group",
			cli:  `cool_secret --group coolGroup -v"cool" -s"production"`,
			wants: SetOpts{
				Key:   "cool_secret",
				Scope: "production",
				Value: "cool",
				Group: "coolGroup",
				Type:  "env_var",
			},
		},
		{
			name: "raw variable with flag",
			cli:  `cool_secret -r -v"$variable_name"`,
			wants: SetOpts{
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
			wants: SetOpts{
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
			wants: SetOpts{
				Key:       "cool_secret",
				Value:     "$variable_name",
				Raw:       false,
				Group:     "",
				Scope:     "*",
				Protected: false,
				Type:      "env_var",
			},
		},
		{
			name: "var with descripton",
			cli:  `var_desc -v"var_desc" -d "cool var description"`,
			wants: SetOpts{
				Key:         "var_desc",
				Protected:   false,
				Value:       "var_desc",
				Group:       "",
				Description: "cool var description",
				Scope:       "*",
				Type:        "env_var",
			},
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

			var gotOpts *SetOpts
			cmd := NewCmdSet(f, func(opts *SetOpts) error {
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
			assert.Equal(t, tt.wants.Scope, gotOpts.Scope)
			assert.Equal(t, tt.wants.Protected, gotOpts.Protected)
			assert.Equal(t, tt.wants.Description, gotOpts.Description)
			assert.Equal(t, tt.wants.Raw, gotOpts.Raw)
			assert.Equal(t, tt.wants.Masked, gotOpts.Masked)
			assert.Equal(t, tt.wants.Type, gotOpts.Type)
		})
	}
}

func Test_setRun_project(t *testing.T) {
	reg := &httpmock.Mocker{}
	defer reg.Verify(t)

	reg.RegisterResponder(http.MethodPost, "/projects/owner/repo/variables",
		httpmock.NewStringResponse(http.StatusCreated, `
			{
    			"key": "NEW_VARIABLE",
    			"value": "new value",
    			"variable_type": "env_var",
    			"protected": false,
    			"masked": false,
				"raw": false,
				"scope": "production"
			}
		`),
	)

	io, _, stdout, _ := iostreams.Test()

	opts := &SetOpts{
		HTTPClient: func() (*gitlab.Client, error) {
			a, _ := api.TestClient(&http.Client{Transport: reg}, "", "gitlab.com", false)
			return a.Lab(), nil
		},
		BaseRepo: func() (glrepo.Interface, error) {
			return glrepo.FromFullName("owner/repo")
		},
		IO:    io,
		Key:   "NEW_VARIABLE",
		Value: "new value",
		Scope: "*",
	}
	_, _ = opts.HTTPClient()

	err := setRun(opts)
	assert.NoError(t, err)
	assert.Equal(t, stdout.String(), "✓ Created variable NEW_VARIABLE for owner/repo with scope *.\n")
}

func Test_setRun_group(t *testing.T) {
	reg := &httpmock.Mocker{}
	defer reg.Verify(t)

	reg.RegisterResponder(http.MethodPost, "/groups/mygroup/variables",
		httpmock.NewStringResponse(http.StatusCreated, `
			{
    			"key": "NEW_VARIABLE",
    			"value": "new value",
    			"variable_type": "env_var",
    			"protected": false,
    			"masked": false,
				"raw": false,
				"scope": "production"
			}
		`),
	)

	io, _, stdout, _ := iostreams.Test()

	opts := &SetOpts{
		HTTPClient: func() (*gitlab.Client, error) {
			a, _ := api.TestClient(&http.Client{Transport: reg}, "", "gitlab.com", false)
			return a.Lab(), nil
		},
		BaseRepo: func() (glrepo.Interface, error) {
			return glrepo.FromFullName("owner/repo")
		},
		IO:    io,
		Key:   "NEW_VARIABLE",
		Value: "new value",
		Group: "mygroup",
	}
	_, _ = opts.HTTPClient()

	err := setRun(opts)
	assert.NoError(t, err)
	assert.Equal(t, stdout.String(), "✓ Created variable NEW_VARIABLE for group mygroup.\n")
}
