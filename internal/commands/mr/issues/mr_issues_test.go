package issues

import (
	"net/http"
	"regexp"
	"testing"

	"github.com/MakeNowJust/heredoc/v2"

	"github.com/stretchr/testify/assert"
	"gitlab.com/gitlab-org/cli/internal/testing/cmdtest"
	"gitlab.com/gitlab-org/cli/internal/testing/httpmock"
	"gitlab.com/gitlab-org/cli/test"
)

func runCommand(t *testing.T, rt http.RoundTripper, cli string) (*test.CmdOut, error) {
	ios, _, stdout, stderr := cmdtest.TestIOStreams(cmdtest.WithTestIOStreamsAsTTY(true))
	factory := cmdtest.NewTestFactory(ios,
		cmdtest.WithGitLabClient(cmdtest.NewTestApiClient(t, &http.Client{Transport: rt}, "", "gitlab.com").Lab()),
		cmdtest.WithBranch("current_branch"),
	)

	cmd := NewCmdIssues(factory)

	return cmdtest.ExecuteCommand(cmd, cli, stdout, stderr)
}

func TestMergeRequestClosesIssues_byID(t *testing.T) {
	// NOTE: we need to force disable colors, otherwise we'd need ANSI sequences in our test output assertions.
	t.Setenv("NO_COLOR", "true")

	fakeHTTP := httpmock.New()
	defer fakeHTTP.Verify(t)

	fakeHTTP.RegisterResponder(http.MethodGet, "/projects/OWNER/REPO/merge_requests/123",
		httpmock.NewStringResponse(http.StatusOK, `
				{
		  			"id": 123,
		  			"iid": 123,
					"web_url": "https://gitlab.com/OWNER/REPO/merge_requests/123"
				}
			`))

	fakeHTTP.RegisterResponder(http.MethodGet, "/projects/OWNER/REPO/merge_requests/123/closes_issues",
		httpmock.NewFileResponse(http.StatusOK, "./testdata/closesIssuesList.json"))

	cli := "123"
	output, err := runCommand(t, fakeHTTP, cli)
	if err != nil {
		t.Errorf("error running command `mr issues %s`: %v", cli, err)
	}

	out := output.String()
	timeRE := regexp.MustCompile(`\d+ years`)
	out = timeRE.ReplaceAllString(out, "X years")

	assert.Equal(t, heredoc.Doc(`
		Showing 2 issues in OWNER/REPO that match your search. 

		#11	new issue                		about X years ago
		#15	this is another new issue		about X years ago

	`), out)
	assert.Equal(t, ``, output.Stderr())
}

func TestMergeRequestClosesIssues_currentBranch(t *testing.T) {
	// NOTE: we need to force disable colors, otherwise we'd need ANSI sequences in our test output assertions.
	t.Setenv("NO_COLOR", "true")

	fakeHTTP := &httpmock.Mocker{
		MatchURL: httpmock.PathAndQuerystring,
	}

	defer fakeHTTP.Verify(t)

	fakeHTTP.RegisterResponder(http.MethodGet, "/api/v4/projects/OWNER/REPO/merge_requests?per_page=30&source_branch=current_branch",
		httpmock.NewStringResponse(http.StatusOK, `
				[{
					"id":123,
					"iid":123,
					"project_id":1,
					"web_url":"https://gitlab.com/OWNER/REPO/merge_requests/123"
				}]
			`))

	fakeHTTP.RegisterResponder(http.MethodGet, "/api/v4/projects/OWNER/REPO/merge_requests/123",
		httpmock.NewStringResponse(http.StatusOK, `
					{
			  			"id": 123,
			  			"iid": 123,
						"web_url": "https://gitlab.com/OWNER/REPO/merge_requests/123"
					}
				`))

	fakeHTTP.RegisterResponder(http.MethodGet, "/api/v4/projects/OWNER/REPO/merge_requests/123/closes_issues",
		httpmock.NewFileResponse(http.StatusOK, "./testdata/closesIssuesList.json"))

	output, err := runCommand(t, fakeHTTP, "")
	if err != nil {
		t.Errorf("error running command `mr issues`: %v", err)
	}

	out := output.String()
	timeRE := regexp.MustCompile(`\d+ years`)
	out = timeRE.ReplaceAllString(out, "X years")

	assert.Equal(t, heredoc.Doc(`
		Showing 2 issues in OWNER/REPO that match your search. 

		#11	new issue                		about X years ago
		#15	this is another new issue		about X years ago

	`), out)
	assert.Equal(t, ``, output.Stderr())
}
