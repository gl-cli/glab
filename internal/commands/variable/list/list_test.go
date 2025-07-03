package list

import (
	"bytes"
	"testing"

	"github.com/google/shlex"
	"github.com/stretchr/testify/assert"
	"gitlab.com/gitlab-org/cli/internal/testing/cmdtest"
)

func Test_NewCmdList(t *testing.T) {
	tests := []struct {
		name     string
		cli      string
		wants    options
		stdinTTY bool
		wantsErr bool
	}{
		{
			name:     "no input",
			cli:      "",
			wantsErr: false,
			wants: options{
				group:        "",
				outputFormat: "text",
				perPage:      20,
				page:         1,
			},
		},
		{
			name:     "no input with json output format",
			cli:      "-F json",
			wantsErr: false,
			wants: options{
				group:        "",
				outputFormat: "json",
				perPage:      20,
				page:         1,
			},
		},
		{
			name:     "group vars",
			cli:      "--group group/group",
			wantsErr: false,
			wants: options{
				group:        "group/group",
				outputFormat: "text",
				perPage:      20,
				page:         1,
			},
		},
		{
			name:     "per page",
			cli:      "--per-page 100 --page 1",
			wantsErr: false,
			wants: options{
				group:        "",
				outputFormat: "text",
				page:         1,
				perPage:      100,
			},
		},
		{
			name:     "instance vars",
			cli:      "--instance",
			wantsErr: false,
			wants: options{
				instance:     true,
				outputFormat: "text",
				perPage:      20,
				page:         1,
			},
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
			cmd := NewCmdList(f, func(opts *options) error {
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

			assert.Equal(t, tt.wants.group, gotOpts.group)
			assert.Equal(t, tt.wants.outputFormat, gotOpts.outputFormat)
			assert.Equal(t, tt.wants.page, gotOpts.page)
			assert.Equal(t, tt.wants.perPage, gotOpts.perPage)
		})
	}
}
