package list

import (
	"net/http"
	"testing"

	"github.com/MakeNowJust/heredoc/v2"

	"github.com/stretchr/testify/assert"
	"gitlab.com/gitlab-org/cli/internal/glinstance"
	"gitlab.com/gitlab-org/cli/internal/testing/cmdtest"
	"gitlab.com/gitlab-org/cli/internal/testing/httpmock"
	"gitlab.com/gitlab-org/cli/test"
)

func runCommand(t *testing.T, rt http.RoundTripper, cli string) (*test.CmdOut, error) {
	ios, _, stdout, stderr := cmdtest.TestIOStreams(cmdtest.WithTestIOStreamsAsTTY(true))
	tc := cmdtest.NewTestApiClient(t, &http.Client{Transport: rt}, "", glinstance.DefaultHostname)
	factory := cmdtest.NewTestFactory(ios,
		cmdtest.WithApiClient(tc),
		cmdtest.WithGitLabClient(tc.Lab()),
	)
	cmd := NewCmdList(factory)
	return cmdtest.ExecuteCommand(cmd, cli, stdout, stderr)
}

func TestLabelList(t *testing.T) {
	fakeHTTP := &httpmock.Mocker{}
	defer fakeHTTP.Verify(t)

	fakeHTTP.RegisterResponder(http.MethodGet, "/api/v4/projects/OWNER/REPO/labels",
		httpmock.NewStringResponse(http.StatusOK, `
	[
		{
			"id":1,
			"name":"bug",
			"description":null,
			"text_color":"#FFFFFF",
			"color":"#6699cc",
			"priority":null,
			"is_project_label":true
		},
		{
			"id":2,
			"name":"ux",
			"description":"User Experience",
			"text_color":"#FFFFFF",
			"color":"#3cb371",
			"priority":null,
			"is_project_label":true
		}
	]
	`))

	output, err := runCommand(t, fakeHTTP, "")
	if err != nil {
		t.Errorf("error running command `label list`: %v", err)
	}

	out := output.String()

	assert.Equal(t, heredoc.Doc(`
		Showing label 2 of 2 on OWNER/REPO.

		 bug (#6699cc)
		 ux -> User Experience (#3cb371)
 
	`), out)
	assert.Empty(t, output.Stderr())
}

func TestLabelListJSON(t *testing.T) {
	fakeHTTP := httpmock.New()
	fakeHTTP.MatchURL = httpmock.PathAndQuerystring
	defer fakeHTTP.Verify(t)

	expectedBody := `[
    {
        "id": 29739671,
        "name": "my label",
        "color": "#00b140",
        "text_color": "#FFFFFF",
        "description": "Simple label",
        "open_issues_count": 0,
        "closed_issues_count": 0,
        "open_merge_requests_count": 0,
        "subscribed": false,
        "priority": 0,
        "is_project_label": true
    }
]`

	fakeHTTP.RegisterResponder(http.MethodGet, "/api/v4/projects/OWNER%2FREPO/labels?page=1&per_page=30&with_counts=true",
		httpmock.NewStringResponse(http.StatusOK, `[
  {
    "id": 29739671,
    "name": "my label",
    "description": "Simple label",
    "description_html": "Simple label",
    "text_color": "#FFFFFF",
    "color": "#00b140",
    "open_issues_count": 0,
    "closed_issues_count": 0,
    "open_merge_requests_count": 0,
    "subscribed": false,
    "priority": null,
    "is_project_label": true
  }
]`))

	output, err := runCommand(t, fakeHTTP, "-F json")
	if err != nil {
		t.Errorf("error running command `label list -F json`: %v", err)
	}

	assert.JSONEq(t, expectedBody, output.String())
	assert.Empty(t, output.Stderr())
}

func TestGroupLabelList(t *testing.T) {
	fakeHTTP := &httpmock.Mocker{}
	defer fakeHTTP.Verify(t)

	fakeHTTP.RegisterResponder(http.MethodGet, "/api/v4/groups/foo/labels",
		httpmock.NewStringResponse(http.StatusOK, `
	[
		{
			"id":1,
			"name":"groupbug",
			"description":null,
			"text_color":"#FFFFFF",
			"color":"#6699cc",
			"priority":null,
			"is_project_label":false
		},
		{
			"id":2,
			"name":"groupux",
			"description":"User Experience",
			"text_color":"#FFFFFF",
			"color":"#3cb371",
			"priority":null,
			"is_project_label":false
		}
	]
	`))

	flags := "--group foo"
	output, err := runCommand(t, fakeHTTP, flags)
	if err != nil {
		t.Errorf("error running command `label list %s`: %v", flags, err)
	}

	out := output.String()

	assert.Equal(t, heredoc.Doc(`
		Showing label 2 of 2 for group foo.

		 groupbug (#6699cc)
		 groupux -> User Experience (#3cb371)
 
	`), out)
	assert.Empty(t, output.Stderr())
}
