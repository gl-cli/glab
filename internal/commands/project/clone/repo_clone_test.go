package clone

import (
	"bytes"
	"testing"

	"github.com/spf13/cobra"

	"github.com/google/shlex"
	"github.com/stretchr/testify/require"
	"gitlab.com/gitlab-org/cli/test"

	"github.com/stretchr/testify/assert"
	"gitlab.com/gitlab-org/cli/internal/testing/cmdtest"
)

func TestMain(m *testing.M) {
	cmdtest.InitTest(m, "repo_clone_test")
}

func runCommand(cmd *cobra.Command, cli string, stds ...*bytes.Buffer) (*test.CmdOut, error) {
	var stdin *bytes.Buffer
	var stderr *bytes.Buffer
	var stdout *bytes.Buffer

	for i, std := range stds {
		if std != nil {
			if i == 0 {
				stdin = std
			}
			if i == 1 {
				stdout = std
			}
			if i == 2 {
				stderr = std
			}
		}
	}
	cmd.SetIn(stdin)
	cmd.SetOut(stdout)
	cmd.SetErr(stderr)

	argv, err := shlex.Split(cli)
	if err != nil {
		return nil, err
	}
	cmd.SetArgs(argv)
	_, err = cmd.ExecuteC()

	return &test.CmdOut{
		OutBuf: stdout,
		ErrBuf: stderr,
	}, err
}

func TestNewCmdClone(t *testing.T) {
	testCases := []struct {
		name        string
		args        string
		wantOpts    options
		wantCtxOpts ContextOpts
		wantErr     string
	}{
		{
			name:    "no arguments",
			args:    "",
			wantErr: "Specify repository argument, or use the --group flag to specify a group to clone all repos from the group.",
		},
		{
			name: "repo argument",
			args: "NAMESPACE/REPO",
			wantOpts: options{
				gitFlags: []string{},
			},
			wantCtxOpts: ContextOpts{
				Repo: "NAMESPACE/REPO",
			},
		},
		{
			name: "directory argument",
			args: "NAMESPACE/REPO mydir",
			wantOpts: options{
				gitFlags: []string{},
				dir:      "mydir",
			},
			wantCtxOpts: ContextOpts{
				Repo: "NAMESPACE/REPO",
			},
		},
		{
			name: "git clone arguments",
			args: "NAMESPACE/REPO -- --depth 1 --recurse-submodules",
			wantOpts: options{
				gitFlags: []string{"--depth", "1", "--recurse-submodules"},
			},
			wantCtxOpts: ContextOpts{
				Repo: "NAMESPACE/REPO",
			},
		},
		{
			name: "group clone arguments",
			args: "-g NAMESPACE/REPO -- --depth 1 --recurse-submodules",
			wantOpts: options{
				gitFlags: []string{"--depth", "1", "--recurse-submodules"},
			},
			wantCtxOpts: ContextOpts{
				Repo: "",
			},
		},
		{
			name:    "unknown argument",
			args:    "NAMESPACE/REPO --depth 1",
			wantErr: "unknown flag: --depth\nSeparate Git clone flags with '--'.",
		},
	}
	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			io, stdin, stdout, stderr := cmdtest.TestIOStreams()
			fac := cmdtest.NewTestFactory(io)

			var opts *options
			var ctxOpts *ContextOpts
			cmd := NewCmdClone(fac, func(co *options, cx *ContextOpts) error {
				opts = co
				ctxOpts = cx
				return nil
			})

			argv, err := shlex.Split(tt.args)
			require.NoError(t, err)
			cmd.SetArgs(argv)

			cmd.SetIn(stdin)
			cmd.SetOut(stdout)
			cmd.SetErr(stderr)

			_, err = cmd.ExecuteC()
			if err != nil {
				assert.Equal(t, tt.wantErr, err.Error())
				return
			} else if tt.wantErr != "" {
				t.Errorf("expected error %q, got nil", tt.wantErr)
			}

			assert.Equal(t, "", stdout.String())
			assert.Equal(t, "", stderr.String())

			assert.Equal(t, tt.wantCtxOpts.Repo, ctxOpts.Repo)
			assert.Equal(t, tt.wantOpts.gitFlags, opts.gitFlags)
		})
	}
}
