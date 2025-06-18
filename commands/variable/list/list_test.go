package list

import (
	"bytes"
	"testing"

	"github.com/google/shlex"
	"github.com/stretchr/testify/assert"
	"gitlab.com/gitlab-org/cli/commands/cmdtest"
	"gitlab.com/gitlab-org/cli/pkg/iostreams"
)

func Test_NewCmdList(t *testing.T) {
	tests := []struct {
		name     string
		cli      string
		wants    ListOpts
		stdinTTY bool
		wantsErr bool
	}{
		{
			name:     "no input",
			cli:      "",
			wantsErr: false,
			wants: ListOpts{
				Group:        "",
				OutputFormat: "text",
				PerPage:      20,
				Page:         1,
			},
		},
		{
			name:     "no input with json output format",
			cli:      "-F json",
			wantsErr: false,
			wants: ListOpts{
				Group:        "",
				OutputFormat: "json",
				PerPage:      20,
				Page:         1,
			},
		},
		{
			name:     "group vars",
			cli:      "--group group/group",
			wantsErr: false,
			wants: ListOpts{
				Group:        "group/group",
				OutputFormat: "text",
				PerPage:      20,
				Page:         1,
			},
		},
		{
			name:     "per page",
			cli:      "--per-page 100 --page 1",
			wantsErr: false,
			wants: ListOpts{
				Group:        "",
				OutputFormat: "text",
				Page:         1,
				PerPage:      100,
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

			var gotOpts *ListOpts
			cmd := NewCmdList(f, func(opts *ListOpts) error {
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

			assert.Equal(t, tt.wants.Group, gotOpts.Group)
			assert.Equal(t, tt.wants.OutputFormat, gotOpts.OutputFormat)
			assert.Equal(t, tt.wants.Page, gotOpts.Page)
			assert.Equal(t, tt.wants.PerPage, gotOpts.PerPage)
		})
	}
}
