package close

import (
	"fmt"
	"testing"
	"time"

	"gitlab.com/gitlab-org/cli/test"

	"gitlab.com/gitlab-org/cli/pkg/iostreams"

	"github.com/acarl005/stripansi"
	"github.com/stretchr/testify/require"
	"github.com/xanzy/go-gitlab"
	"gitlab.com/gitlab-org/cli/api"

	"github.com/stretchr/testify/assert"
	"gitlab.com/gitlab-org/cli/commands/cmdtest"
)

func Test_issueClose_Integration(t *testing.T) {
	glTestHost := test.GetHostOrSkip(t)

	t.Parallel()

	oldUpdateIssue := api.UpdateIssue
	timer, _ := time.Parse(time.RFC3339, "2014-11-12T11:45:26.371Z")
	api.UpdateIssue = func(client *gitlab.Client, projectID interface{}, issueID int, opts *gitlab.UpdateIssueOptions) (*gitlab.Issue, error) {
		if projectID == "" || projectID == "WRONG_REPO" || projectID == "expected_err" || issueID == 0 {
			return nil, fmt.Errorf("error expected")
		}
		return &gitlab.Issue{
			ID:          issueID,
			IID:         issueID,
			State:       "closed",
			Description: "Dummy description for issue " + string(rune(issueID)),
			Author: &gitlab.IssueAuthor{
				ID:       1,
				Name:     "John Dev Wick",
				Username: "jdwick",
			},
			CreatedAt: &timer,
		}, nil
	}

	testCases := []struct {
		Name        string
		Issue       string
		ExpectedMsg []string
		wantErr     bool
	}{
		{
			Name:        "Issue Exists",
			Issue:       "1",
			ExpectedMsg: []string{"Closing Issue...", "Closed Issue #1"},
		},
		{
			Name:        "Issue Does Not Exist",
			Issue:       "0",
			ExpectedMsg: []string{"Closing Issue", "404 Not found"},
			wantErr:     true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.Name, func(t *testing.T) {
			io, _, stdout, stderr := iostreams.Test()
			f := cmdtest.StubFactory(glTestHost + "/glab-cli/test")
			f.IO = io
			f.IO.IsaTTY = true
			f.IO.IsErrTTY = true
			cmd := NewCmdClose(f)

			cmd.SetArgs([]string{tc.Issue})
			cmd.SetOut(stdout)
			cmd.SetErr(stderr)

			_, err := cmd.ExecuteC()
			if tc.wantErr {
				require.Error(t, err)
				return
			} else {
				require.NoError(t, err)
			}

			out := stripansi.Strip(stdout.String())

			for _, msg := range tc.ExpectedMsg {
				assert.Contains(t, out, msg)
			}
		})
	}

	api.UpdateIssue = oldUpdateIssue
}
