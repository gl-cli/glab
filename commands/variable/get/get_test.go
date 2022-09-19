package get

import (
	"bytes"
	"github.com/alecthomas/assert"
	"github.com/google/shlex"
	"github.com/profclems/glab/api"
	"github.com/profclems/glab/commands/cmdutils"
	"github.com/profclems/glab/internal/glrepo"
	"github.com/profclems/glab/pkg/httpmock"
	"github.com/profclems/glab/pkg/iostreams"
	"github.com/xanzy/go-gitlab"
	"net/http"
	"testing"
)

func Test_NewCmdSet(t *testing.T) {
	tests := []struct {
		name     string
		cli      string
		wants    GetOps
		wantsErr bool
	}{
		{
			name:     "good key",
			cli:      "good_key",
			wantsErr: false,
			wants: GetOps{
				Key: "good_key",
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
			wants: GetOps{
				Key:   "good_key",
				Group: "group",
			},
			wantsErr: false,
		},
		{
			name: "bad key",
			cli:  "-g group bad-key",
			wants: GetOps{
				Group: "group",
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
			f := &cmdutils.Factory{
				IO: io,
			}

			argv, err := shlex.Split(test.cli)
			assert.NoError(t, err)

			var gotOpts *GetOps
			cmd := NewCmdSet(f, func(opts *GetOps) error {
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

			assert.Equal(t, test.wants.Key, gotOpts.Key)
			assert.Equal(t, test.wants.Group, gotOpts.Group)
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

	reg.RegisterResponder("GET", "/projects/owner/repo/variables/TEST_VAR",
		httpmock.NewJSONResponse(200, body),
	)

	io, _, stdout, _ := iostreams.Test()

	opts := &GetOps{
		HTTPClient: func() (*gitlab.Client, error) {
			a, _ := api.TestClient(&http.Client{Transport: reg}, "", "gitlab.com", false)
			return a.Lab(), nil
		},
		BaseRepo: func() (glrepo.Interface, error) {
			return glrepo.FromFullName("owner/repo")
		},
		IO:  io,
		Key: "TEST_VAR",
	}
	_, _ = opts.HTTPClient()

	err := getRun(opts)
	assert.NoError(t, err)
	assert.Equal(t, varContent, stdout.String())
}
