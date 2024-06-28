package help

import (
	"os"
	"testing"

	"gitlab.com/gitlab-org/cli/pkg/iostreams"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"gitlab.com/gitlab-org/cli/commands/alias"
	"gitlab.com/gitlab-org/cli/commands/alias/set"
	"gitlab.com/gitlab-org/cli/commands/cmdutils"
	"gitlab.com/gitlab-org/cli/test"
)

func TestDedent(t *testing.T) {
	type c struct {
		input    string
		expected string
	}

	cases := []c{
		{
			input:    "      --help      Show help for command.\n      --version   Show glab version\n",
			expected: "--help      Show help for command.\n--version   Show glab version\n",
		},
		{
			input:    "      --help              Show help for command.\n  -R, --repo OWNER/REPO   Select another repository using the OWNER/REPO format\n",
			expected: "    --help              Show help for command.\n-R, --repo OWNER/REPO   Select another repository using the OWNER/REPO format\n",
		},
		{
			input:    "  line 1\n\n  line 2\n line 3",
			expected: " line 1\n\n line 2\nline 3",
		},
		{
			input:    "  line 1\n  line 2\n  line 3\n\n",
			expected: "line 1\nline 2\nline 3\n\n",
		},
		{
			input:    "\n\n\n\n\n\n",
			expected: "\n\n\n\n\n\n",
		},
		{
			input:    "",
			expected: "",
		},
	}

	for _, tt := range cases {
		got := dedent(tt.input)
		if got != tt.expected {
			t.Errorf("expected: %q, got: %q", tt.expected, got)
		}
	}
}

func TestRootHelpFunc(t *testing.T) {
	type args struct {
		command *cobra.Command
		args    []string
	}
	tests := []struct {
		name    string
		args    args
		wantOut string
	}{
		{
			name: "alias",
			args: args{
				command: alias.NewCmdAlias(&cmdutils.Factory{}),
			},
			wantOut: `Create, list, and delete aliases.

USAGE
  alias [command] [flags]`,
		},

		{
			name: "test nested alias cmd",
			args: args{
				command: set.NewCmdSet(&cmdutils.Factory{}, nil),
				args:    []string{"set", "-h"},
			},
			wantOut: "USAGE\n  alias set <alias name> '<command>' [flags]\n\nFLAGS\n  -s, --shell ",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			streams, _, _, _ := iostreams.Test()
			old := os.Stdout // keep backup of the real stdout
			r, w, _ := os.Pipe()
			os.Stdout = w
			cmd := tt.args.command
			if len(tt.args.args) > 0 {
				// falsify a parent command
				alias.NewCmdAlias(&cmdutils.Factory{}).AddCommand(cmd)
			}
			RootHelpFunc(streams.Color(), cmd, tt.args.args)

			out := test.ReturnBuffer(old, r, w)

			assert.Contains(t, out, tt.wantOut)
		})
	}
}

func TestRootUsageFunc(t *testing.T) {
	type args struct {
		command *cobra.Command
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			args: args{
				command: alias.NewCmdAlias(&cmdutils.Factory{}),
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := RootUsageFunc(tt.args.command); (err != nil) != tt.wantErr {
				t.Errorf("RootUsageFunc() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
