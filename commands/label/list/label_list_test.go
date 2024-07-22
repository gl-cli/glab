package list

import (
	"net/http"
	"testing"

	"github.com/MakeNowJust/heredoc/v2"

	"github.com/stretchr/testify/assert"
	"gitlab.com/gitlab-org/cli/commands/cmdtest"
	"gitlab.com/gitlab-org/cli/pkg/httpmock"
	"gitlab.com/gitlab-org/cli/test"
)

func runCommand(rt http.RoundTripper, cli string) (*test.CmdOut, error) {
	ios, _, stdout, stderr := cmdtest.InitIOStreams(true, "")
	factory := cmdtest.InitFactory(ios, rt)

	// TODO: shouldn't be there but the stub doesn't work without it
	_, _ = factory.HttpClient()

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

	output, err := runCommand(fakeHTTP, "")
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

	output, err := runCommand(fakeHTTP, "-F json")
	if err != nil {
		t.Errorf("error running command `label list -F json`: %v", err)
	}

	assert.JSONEq(t, expectedBody, output.String())
	assert.Empty(t, output.Stderr())
}
