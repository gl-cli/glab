package export

import (
	"bytes"
	"net/http"
	"testing"

	"github.com/google/shlex"
	"github.com/stretchr/testify/assert"
	"github.com/xanzy/go-gitlab"
	"gitlab.com/gitlab-org/cli/api"
	"gitlab.com/gitlab-org/cli/commands/cmdutils"
	"gitlab.com/gitlab-org/cli/internal/glrepo"
	"gitlab.com/gitlab-org/cli/pkg/httpmock"
	"gitlab.com/gitlab-org/cli/pkg/iostreams"
)

func Test_NewCmdExport(t *testing.T) {
	tests := []struct {
		name     string
		cli      string
		wants    ExportOpts
		wantsErr bool
	}{
		{
			name:     "no arguments",
			cli:      "",
			wantsErr: false,
		},
		{
			name:     "with group",
			cli:      "--group STH",
			wantsErr: false,
			wants: ExportOpts{
				Group: "STH",
			},
		},
		{
			name:     "missing group",
			cli:      "--group",
			wantsErr: true,
			wants: ExportOpts{
				Group: "STH",
			},
		},
		{
			name:     "with pagination",
			cli:      "--page 11 --per-page 12",
			wantsErr: false,
		},
		{
			name:     "with invalid pagination",
			cli:      "--page aa --per-page bb",
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

			var gotOpts *ExportOpts
			cmd := NewCmdExport(f, func(opts *ExportOpts) error {
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

			assert.Equal(t, test.wants.Group, gotOpts.Group)
		})
	}
}

func Test_exportRun_project(t *testing.T) {
	reg := &httpmock.Mocker{
		MatchURL: httpmock.FullURL,
	}
	defer reg.Verify(t)

	reg.RegisterResponder(http.MethodGet, "https://gitlab.com/api/v4/projects/owner%2Frepo/variables?page=1&per_page=10",
		httpmock.NewJSONResponse(http.StatusOK, nil),
	)

	io, _, stdout, _ := iostreams.Test()

	opts := &ExportOpts{
		HTTPClient: func() (*gitlab.Client, error) {
			a, _ := api.TestClient(&http.Client{Transport: reg}, "", "gitlab.com", false)
			return a.Lab(), nil
		},
		BaseRepo: func() (glrepo.Interface, error) {
			return glrepo.FromFullName("owner/repo")
		},
		IO:      io,
		Page:    1,
		PerPage: 10,
	}
	_, _ = opts.HTTPClient()

	err := exportRun(opts)
	assert.NoError(t, err)
	assert.Equal(t, "", stdout.String())
}

func Test_exportRun_group(t *testing.T) {
	reg := &httpmock.Mocker{
		MatchURL: httpmock.FullURL,
	}
	defer reg.Verify(t)

	reg.RegisterResponder(http.MethodGet, "https://gitlab.com/api/v4/groups/GROUP/variables?page=7&per_page=77",
		httpmock.NewJSONResponse(http.StatusOK, nil),
	)

	io, _, stdout, _ := iostreams.Test()

	opts := &ExportOpts{
		HTTPClient: func() (*gitlab.Client, error) {
			a, _ := api.TestClient(&http.Client{Transport: reg}, "", "gitlab.com", false)
			return a.Lab(), nil
		},
		BaseRepo: func() (glrepo.Interface, error) {
			return glrepo.FromFullName("owner/repo")
		},
		IO:      io,
		Page:    7,
		PerPage: 77,
		Group:   "GROUP",
	}
	_, _ = opts.HTTPClient()

	err := exportRun(opts)
	assert.NoError(t, err)
	assert.Equal(t, "", stdout.String())
}
