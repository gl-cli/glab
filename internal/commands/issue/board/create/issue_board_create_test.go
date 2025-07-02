package create

import (
	"fmt"
	"testing"

	"gitlab.com/gitlab-org/cli/internal/cmdutils"

	"github.com/acarl005/stripansi"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	gitlab "gitlab.com/gitlab-org/api/client-go"
	"gitlab.com/gitlab-org/cli/internal/testing/cmdtest"
)

func TestNewCmdCreate(t *testing.T) {
	createIssueBoard = func(client *gitlab.Client, projectID any, opts *gitlab.CreateIssueBoardOptions) (*gitlab.IssueBoard, error) {
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

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			exec := cmdtest.SetupCmdForTest(
				t,
				func(f cmdutils.Factory) *cobra.Command {
					cmd := NewCmdCreate(f)
					cmdutils.EnableRepoOverride(cmd, f)
					return cmd
				},
			)

			cmdOut, err := exec(tc.arg)
			if tc.wantErr {
				require.Error(t, err)
				return
			} else {
				require.NoError(t, err)
			}

			out := stripansi.Strip(cmdOut.OutBuf.String())

			assert.Contains(t, out, tc.want)
			assert.Contains(t, cmdOut.ErrBuf.String(), "")
		})
	}
}
