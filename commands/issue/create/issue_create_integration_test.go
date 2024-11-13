package create

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"gitlab.com/gitlab-org/cli/test"

	"gitlab.com/gitlab-org/cli/pkg/iostreams"

	"gitlab.com/gitlab-org/cli/pkg/prompt"

	"gitlab.com/gitlab-org/cli/pkg/utils"

	"github.com/acarl005/stripansi"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/xanzy/go-gitlab"
	"gitlab.com/gitlab-org/cli/api"
	"gitlab.com/gitlab-org/cli/commands/cmdtest"
)

func Test_IssueCreate_Integration(t *testing.T) {
	glTestHost := test.GetHostOrSkip(t)

	cmdtest.CopyTestRepo(t, "issue_create")
	ask, teardown := prompt.InitAskStubber()
	defer teardown()

	ask.Stub([]*prompt.QuestionStub{
		{
			Name:  "confirmation",
			Value: 0,
		},
	})

	oldCreateIssue := api.CreateIssue
	timer, _ := time.Parse(time.RFC3339, "2014-11-12T11:45:26.371Z")
	api.CreateIssue = func(client *gitlab.Client, projectID interface{}, opts *gitlab.CreateIssueOptions) (*gitlab.Issue, error) {
		if projectID == "" || projectID == "WRONG_REPO" || projectID == "expected_err" {
			return nil, fmt.Errorf("error expected")
		}
		return &gitlab.Issue{
			ID:          1,
			IID:         1,
			Title:       *opts.Title,
			Labels:      gitlab.Labels(*opts.Labels),
			State:       "opened",
			Description: *opts.Description,
			Weight:      *opts.Weight,
			Author: &gitlab.IssueAuthor{
				ID:       1,
				Name:     "John Dev Wick",
				Username: "jdwick",
			},
			WebURL:    glTestHost + "/cli-automated-testing/test/-/issues/1",
			CreatedAt: &timer,
		}, nil
	}

	io, _, stdout, stderr := iostreams.Test()
	f := cmdtest.StubFactory(glTestHost + "/cli-automated-testing/test")
	f.IO = io
	f.IO.IsaTTY = true
	f.IO.IsErrTTY = true

	cmd := NewCmdCreate(f)
	cmd.Flags().StringP("repo", "R", "", "")

	cliStr := []string{
		"-t", "myissuetitle",
		"-d", "myissuebody",
		"-l", "test,bug",
		"--weight", "1",
		"--milestone", "1",
		"--linked-mr", "3",
		"--confidential",
		"--assignee", "testuser",
		"-R", "cli-automated-testing/test",
	}

	cli := strings.Join(cliStr, " ")
	_, err := cmdtest.RunCommand(cmd, cli)
	assert.Nil(t, err)

	out := stripansi.Strip(stdout.String())
	outErr := stripansi.Strip(stderr.String())
	expectedOut := fmt.Sprintf("#1 myissuetitle (%s)", utils.TimeToPrettyTimeAgo(timer))
	cmdtest.Eq(t, cmdtest.FirstLine([]byte(out)), expectedOut)
	cmdtest.Eq(t, outErr, "- Creating issue in cli-automated-testing/test\n")
	assert.Contains(t, out, glTestHost+"/cli-automated-testing/test/-/issues/1")

	api.CreateIssue = oldCreateIssue
}

func Test_IssueCreate_With_Recover_Integration(t *testing.T) {
	glTestHost := test.GetHostOrSkip(t)

	cmdtest.CopyTestRepo(t, "issue_create_with_recover")
	ask, teardown := prompt.InitAskStubber()
	defer teardown()

	ask.Stub([]*prompt.QuestionStub{
		{
			Name:  "confirmation",
			Value: 0,
		},
	})

	oldCreateIssue := api.CreateIssue
	timer, _ := time.Parse(time.RFC3339, "2014-11-12T11:45:26.371Z")
	api.CreateIssue = func(client *gitlab.Client, projectID interface{}, opts *gitlab.CreateIssueOptions) (*gitlab.Issue, error) {
		if projectID == "" || projectID == "WRONG_REPO" || projectID == "expected_err" {
			return nil, fmt.Errorf("error expected")
		}
		return &gitlab.Issue{
			ID:          1,
			IID:         1,
			Title:       *opts.Title,
			Labels:      gitlab.Labels(*opts.Labels),
			State:       "opened",
			Description: *opts.Description,
			Weight:      *opts.Weight,
			Author: &gitlab.IssueAuthor{
				ID:       1,
				Name:     "John Dev Wick",
				Username: "jdwick",
			},
			WebURL:    glTestHost + "/cli-automated-testing/test/-/issues/2",
			CreatedAt: &timer,
		}, nil
	}

	io, _, stdout, stderr := iostreams.Test()
	f := cmdtest.StubFactory(glTestHost + "/cli-automated-testing/test")
	f.IO = io
	f.IO.IsaTTY = true
	f.IO.IsErrTTY = true

	oldCreateRun := createRun

	// Force createRun to throw error
	createRun = func(opts *CreateOpts) error {
		return errors.New("fail on purpose")
	}

	cmd := NewCmdCreate(f)
	cmd.Flags().StringP("repo", "R", "", "")

	cliStr := []string{
		"-t", "myissuetitle",
		"-d", "myissuebody",
		"-l", "test,bug",
		"--weight", "1",
		"--milestone", "1",
		"--linked-mr", "3",
		"--confidential",
		"--assignee", "testuser",
		"-R", "cli-automated-testing/test",
	}

	cli := strings.Join(cliStr, " ")
	_, err := cmdtest.RunCommand(cmd, cli)
	assert.Contains(t, err.Error(), "fail on purpose")

	out := stripansi.Strip(stdout.String())
	outErr := stripansi.Strip(stderr.String())

	assert.Contains(t, outErr, "Failed to create issue. Created recovery file: ")
	assert.Empty(t, out)

	// Revert to original state
	createRun = oldCreateRun

	// Run create issue with recover
	newCliStr := append(cliStr, "--recover")

	stdout.Reset()
	stderr.Reset()

	newcli := strings.Join(newCliStr, " ")

	_, newerr := cmdtest.RunCommand(cmd, newcli)
	assert.Nil(t, newerr)

	newout := stripansi.Strip(stdout.String())
	newoutErr := stripansi.Strip(stderr.String())
	expectedOut := fmt.Sprintf("#1 myissuetitle (%s)", utils.TimeToPrettyTimeAgo(timer))

	assert.Contains(t, newout, expectedOut)
	assert.Contains(t, newout, "Recovered create options from file.")
	cmdtest.Eq(t, newoutErr, "- Creating issue in cli-automated-testing/test\n")
	assert.Contains(t, newout, glTestHost+"/cli-automated-testing/test/-/issues/2")

	api.CreateIssue = oldCreateIssue
}
