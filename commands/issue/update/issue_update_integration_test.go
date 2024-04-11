package update

import (
	"fmt"
	"io"
	"testing"
	"time"

	"gitlab.com/gitlab-org/cli/test"

	"gitlab.com/gitlab-org/cli/pkg/iostreams"

	"github.com/google/shlex"

	"github.com/acarl005/stripansi"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/xanzy/go-gitlab"
	"gitlab.com/gitlab-org/cli/api"
	"gitlab.com/gitlab-org/cli/commands/cmdtest"
)

func TestNewCmdUpdate_Integration(t *testing.T) {
	glTestHost := test.GetHostOrSkip(t)

	t.Parallel()

	oldUpdateIssue := api.UpdateIssue
	timer, _ := time.Parse(time.RFC3339, "2014-11-12T11:45:26.371Z")
	testIssue := &gitlab.Issue{
		ID:               1,
		IID:              1,
		State:            "closed",
		Labels:           gitlab.Labels{"bug, test, removeable-label"},
		Description:      "Dummy description for issue 1",
		DiscussionLocked: false,
		Author: &gitlab.IssueAuthor{
			ID:       1,
			Name:     "John Dev Wick",
			Username: "jdwick",
		},
		CreatedAt: &timer,
	}
	api.UpdateIssue = func(client *gitlab.Client, projectID interface{}, issueID int, opts *gitlab.UpdateIssueOptions) (*gitlab.Issue, error) {
		if projectID == "" || projectID == "WRONG_REPO" || projectID == "expected_err" || issueID != testIssue.ID {
			return nil, fmt.Errorf("error expected")
		}
		if *opts.Title != "" {
			testIssue.Title = *opts.Title
		}
		if *opts.Description != "" {
			testIssue.Description = *opts.Description
		}
		if opts.AddLabels != nil {
			testIssue.Labels = gitlab.Labels(*opts.AddLabels)
		}
		return testIssue, nil
	}

	testCases := []struct {
		Name        string
		Issue       string
		ExpectedMsg []string
		wantErr     bool
	}{
		{
			Name:  "Issue Exists",
			Issue: `1 -t "New Title" -d "A new description" --lock-discussion -l newLabel --unlabel bug`,
			ExpectedMsg: []string{
				"- Updating issue #1",
				"✓ updated title to \"New Title\"",
				"✓ locked discussion",
				"✓ added labels newLabel",
				"✓ removed labels bug",
				"#1 New Title",
			},
		},
		{
			Name:  "Issue Exists on different repo",
			Issue: `1 -R glab_cli/test`,
			ExpectedMsg: []string{
				"- Updating issue #1",
				"✓ updated title to \"New Title\"",
				"✓ locked discussion",
				"✓ added labels newLabel",
				"✓ removed labels bug",
				"#1 New Title",
			},
		},
		{
			Name:        "Issue Does Not Exist",
			Issue:       "0",
			ExpectedMsg: []string{"- Updating issue #0", "error expected"},
			wantErr:     true,
		},
	}

	ios, _, stdout, stderr := iostreams.Test()
	f := cmdtest.StubFactory(glTestHost + "/cli-automated-testing/test")
	f.IO = ios
	f.IO.IsaTTY = true
	f.IO.IsErrTTY = true

	cmd := NewCmdUpdate(f)
	cmd.Flags().StringP("repo", "R", "", "")

	for _, tc := range testCases {
		t.Run(tc.Name, func(t *testing.T) {
			args, _ := shlex.Split(tc.Issue)
			cmd.SetArgs(args)
			cmd.SetOut(io.Discard)
			cmd.SetErr(io.Discard)

			_, err := cmd.ExecuteC()
			if tc.wantErr {
				require.Error(t, err)
				return
			} else {
				require.NoError(t, err)
			}

			out := stripansi.Strip(stdout.String())
			outErr := stripansi.Strip(stderr.String())

			for _, msg := range tc.ExpectedMsg {
				assert.Contains(t, out, msg)
				assert.Contains(t, outErr, "")
			}
		})
	}

	api.UpdateIssue = oldUpdateIssue
}
