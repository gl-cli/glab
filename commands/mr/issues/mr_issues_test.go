package issues

import (
	"bytes"
	"io"
	"net/http"
	"regexp"
	"testing"

	"gitlab.com/gitlab-org/cli/internal/glrepo"
	"gitlab.com/gitlab-org/cli/pkg/iostreams"

	"github.com/MakeNowJust/heredoc"

	"github.com/alecthomas/assert"
	"github.com/google/shlex"
	"github.com/xanzy/go-gitlab"
	"gitlab.com/gitlab-org/cli/api"
	"gitlab.com/gitlab-org/cli/commands/cmdutils"
	"gitlab.com/gitlab-org/cli/pkg/httpmock"
	"gitlab.com/gitlab-org/cli/test"
)

func runCommand(rt http.RoundTripper, isTTY bool, cli string) (*test.CmdOut, error) {
	ios, _, stdout, stderr := iostreams.Test()
	ios.IsaTTY = isTTY
	ios.IsInTTY = isTTY
	ios.IsErrTTY = isTTY

	factory := &cmdutils.Factory{
		IO: ios,
		HttpClient: func() (*gitlab.Client, error) {
			a, err := api.TestClient(&http.Client{Transport: rt}, "", "", false)
			if err != nil {
				return nil, err
			}
			return a.Lab(), err
		},
		BaseRepo: func() (glrepo.Interface, error) {
			return glrepo.New("OWNER", "REPO"), nil
		},
		Branch: func() (string, error) { return "current_branch", nil },
	}

	// TODO: shouldn't be there but the stub doesn't work without it
	_, _ = factory.HttpClient()

	cmd := NewCmdIssues(factory)

	argv, err := shlex.Split(cli)
	if err != nil {
		return nil, err
	}
	cmd.SetArgs(argv)
	cmd.SetIn(&bytes.Buffer{})
	cmd.SetOut(io.Discard)
	cmd.SetErr(io.Discard)

	_, err = cmd.ExecuteC()
	return &test.CmdOut{
		OutBuf: stdout,
		ErrBuf: stderr,
	}, err
}

func TestMergeRequestClosesIssues_byID(t *testing.T) {
	fakeHTTP := httpmock.New()
	defer fakeHTTP.Verify(t)

	fakeHTTP.RegisterResponder("GET", "/projects/OWNER/REPO/merge_requests/123",
		httpmock.NewStringResponse(200, `
				{
		  			"id": 123,
		  			"iid": 123,
					"web_url": "https://gitlab.com/OWNER/REPO/merge_requests/123"
				}
			`))

	fakeHTTP.RegisterResponder("GET", "/projects/OWNER/REPO/merge_requests/123/closes_issues",
		httpmock.NewFileResponse(200, "./fixtures/closesIssuesList.json"))

	cli := "123"
	output, err := runCommand(fakeHTTP, true, cli)
	if err != nil {
		t.Errorf("error running command `mr issues %s`: %v", cli, err)
	}

	out := output.String()
	timeRE := regexp.MustCompile(`\d+ years`)
	out = timeRE.ReplaceAllString(out, "X years")

	assert.Equal(t, heredoc.Doc(`
		Showing 2 issues in OWNER/REPO that match your search 

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

	fakeHTTP.RegisterResponder("GET", "/api/v4/projects/OWNER/REPO/merge_requests?per_page=30&source_branch=current_branch",
		httpmock.NewStringResponse(200, `
				[{
					"id":123,
					"iid":123,
					"project_id":1,
					"web_url":"https://gitlab.com/OWNER/REPO/merge_requests/123"
				}]
			`))

	fakeHTTP.RegisterResponder("GET", "/api/v4/projects/OWNER/REPO/merge_requests/123",
		httpmock.NewStringResponse(200, `
					{
			  			"id": 123,
			  			"iid": 123,
						"web_url": "https://gitlab.com/OWNER/REPO/merge_requests/123"
					}
				`))

	fakeHTTP.RegisterResponder("GET", "/api/v4/projects/OWNER/REPO/merge_requests/123/closes_issues",
		httpmock.NewFileResponse(200, "./fixtures/closesIssuesList.json"))

	output, err := runCommand(fakeHTTP, true, "")
	if err != nil {
		t.Errorf("error running command `mr issues`: %v", err)
	}

	out := output.String()
	timeRE := regexp.MustCompile(`\d+ years`)
	out = timeRE.ReplaceAllString(out, "X years")

	assert.Equal(t, heredoc.Doc(`
		Showing 2 issues in OWNER/REPO that match your search 

		#11	new issue                		about X years ago
		#15	this is another new issue		about X years ago

	`), out)
	assert.Equal(t, ``, output.Stderr())
}
