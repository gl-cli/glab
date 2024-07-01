package issues

import (
	"net/http"
	"regexp"
	"testing"

	"github.com/MakeNowJust/heredoc/v2"

	"github.com/stretchr/testify/assert"
	"gitlab.com/gitlab-org/cli/commands/cmdtest"
	"gitlab.com/gitlab-org/cli/pkg/httpmock"
	"gitlab.com/gitlab-org/cli/test"
)

func runCommand(rt http.RoundTripper, isTTY bool, cli string) (*test.CmdOut, error) {
	ios, _, stdout, stderr := cmdtest.InitIOStreams(isTTY, "")
	factory := cmdtest.InitFactory(ios, rt)
	factory.Branch = func() (string, error) { return "current_branch", nil }

	// TODO: shouldn't be there but the stub doesn't work without it
	_, _ = factory.HttpClient()

	cmd := NewCmdIssues(factory)

	return cmdtest.ExecuteCommand(cmd, cli, stdout, stderr)
}

func TestMergeRequestClosesIssues_byID(t *testing.T) {
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
	output, err := runCommand(fakeHTTP, true, cli)
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

	output, err := runCommand(fakeHTTP, true, "")
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
