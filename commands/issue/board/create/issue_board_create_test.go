package create

import (
	"fmt"
	"testing"

	"gitlab.com/gitlab-org/cli/pkg/iostreams"

	"gitlab.com/gitlab-org/cli/commands/cmdutils"

	"github.com/acarl005/stripansi"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	gitlab "gitlab.com/gitlab-org/api/client-go"
	"gitlab.com/gitlab-org/cli/api"
	"gitlab.com/gitlab-org/cli/commands/cmdtest"
)

func TestNewCmdCreate(t *testing.T) {
	api.CreateIssueBoard = func(client *gitlab.Client, projectID interface{}, opts *gitlab.CreateIssueBoardOptions) (*gitlab.IssueBoard, error) {
		if projectID == "" || projectID == "WRONG_REPO" || projectID == "NS/WRONG_REPO" {
			return nil, fmt.Errorf("error expected")
		}

		return &gitlab.IssueBoard{
			ID:        11,
			Name:      *opts.Name,
			Project:   &gitlab.Project{PathWithNamespace: projectID.(string)},
			Milestone: nil,
			Lists:     nil,
		}, nil
	}
	tests := []struct {
		name    string
		arg     string
		want    string
		wantErr bool
	}{
		{
			name: "Name passed as arg",
			arg:  `"Test"`,
			want: `✓ Board created: "Test"`,
		},
		{
			name: "Name passed in name flag",
			arg:  `--name "Test"`,
			want: `✓ Board created: "Test"`,
		},
		{
			name:    "WRONG_REPO",
			arg:     `"Test" -R NS/WRONG_REPO`,
			wantErr: true,
		},
	}
	io, _, stdout, stderr := iostreams.Test()

	f := cmdtest.StubFactory("https://gitlab.com/cli-automated-testing/test")
	f.IO = io
	f.IO.IsaTTY = true
	f.IO.IsErrTTY = true

	cmd := NewCmdCreate(f)
	cmdutils.EnableRepoOverride(cmd, f)

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, err := cmdtest.RunCommand(cmd, tc.arg)
			if tc.wantErr {
				require.Error(t, err)
				return
			} else {
				require.NoError(t, err)
			}

			out := stripansi.Strip(stdout.String())

			assert.Contains(t, out, tc.want)
			assert.Contains(t, stderr.String(), "")
		})
	}
}
