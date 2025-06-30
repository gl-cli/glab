package archive

import (
	"strings"
	"testing"

	"gitlab.com/gitlab-org/cli/test"

	"github.com/stretchr/testify/assert"
	"gitlab.com/gitlab-org/cli/internal/commands/cmdtest"
)

func runCommand(cli string) (*test.CmdOut, error) {
	ios, _, stdout, stderr := cmdtest.TestIOStreams()
	factory := cmdtest.InitFactory(ios, nil)

	cmd := NewCmdArchive(factory)

	return cmdtest.ExecuteCommand(cmd, cli, stdout, stderr)
}

func Test_repoArchive_Integration(t *testing.T) {
	glTestHost := test.GetHostOrSkip(t)
	t.Setenv("GITLAB_HOST", glTestHost)

	type argFlags struct {
		format string
		sha    string
		repo   string
		dest   string
	}

	tests := []struct {
		name    string
		args    argFlags
		wantMsg string
		wantErr bool
	}{
		{
			name:    "Has invalid format",
			args:    argFlags{"asp", "master", "cli-automated-testing/test", "test"},
			wantMsg: "format must be one of",
			wantErr: true,
		},
		{
			name:    "Has valid format",
			args:    argFlags{"zip", "master", "cli-automated-testing/test", "test"},
			wantMsg: "Complete... test.zip",
		},
		{
			name:    "Repo is invalid",
			args:    argFlags{"zip", "master", "cli-automated-testing/testzz", "test"},
			wantMsg: "404 Not Found",
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmdArgs := []string{tt.args.repo, tt.args.dest, "--format", tt.args.format, "--sha", tt.args.sha}
			out, err := runCommand(strings.Join(cmdArgs, " "))
			if err != nil {
				t.Log(err)
				if !tt.wantErr {
					t.Fatal(err)
				}
			}

			if tt.wantErr {
				assert.Contains(t, err.Error(), tt.wantMsg)
			} else {
				assert.Contains(t, out.String(), tt.wantMsg)
			}
		})
	}
}
