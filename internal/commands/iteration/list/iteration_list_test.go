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
	factory := cmdtest.NewTestFactory(ios,
		cmdtest.WithApiClient(cmdtest.NewTestApiClient(t, &http.Client{Transport: rt}, "", glinstance.DefaultHostname)),
	)

	cmd := NewCmdList(factory)

	return cmdtest.ExecuteCommand(cmd, cli, stdout, stderr)
}

func TestIterationList(t *testing.T) {
	fakeHTTP := &httpmock.Mocker{}
	defer fakeHTTP.Verify(t)

	fakeHTTP.RegisterResponder(http.MethodGet, "/api/v4/projects/OWNER/REPO/iterations",
		httpmock.NewStringResponse(http.StatusOK, `
	[
  		{
			"id": 53,
			"iid": 13,
			"group_id": 5,
			"title": "Iteration II",
			"description": "Ipsum Lorem ipsum",
			"state": 2,
			"created_at": "2020-01-27T05:07:12.573Z",
			"updated_at": "2020-01-27T05:07:12.573Z",
			"due_date": "2020-02-01",
			"start_date": "2020-02-14",
			"web_url": "http://gitlab.example.com/groups/my-group/-/iterations/13"
  		}
	]
	`))

	output, err := runCommand(t, fakeHTTP, "")
	if err != nil {
		t.Errorf("error running command `iteration list`: %v", err)
	}
	out := output.String()
	assert.Equal(t, heredoc.Doc(`
		Showing iteration 1 of 1 on OWNER/REPO.

		 Iteration II -> Ipsum Lorem ipsum (http://gitlab.example.com/groups/my-group/-/iterations/13)
 
	`), out)
	assert.Empty(t, output.Stderr())
}

func TestIterationListJSON(t *testing.T) {
	fakeHTTP := httpmock.New()
	fakeHTTP.MatchURL = httpmock.PathAndQuerystring
	defer fakeHTTP.Verify(t)

	expectedBody := `[
  {
    "id": 53,
    "iid": 13,
    "group_id": 5,
    "title": "Iteration II",
    "description": "Ipsum Lorem ipsum",
    "state": 2,
    "created_at": "2020-01-27T05:07:12.573Z",
    "updated_at": "2020-01-27T05:07:12.573Z",
    "due_date": "2020-02-01",
    "start_date": "2020-02-14",
	"sequence": 0,
    "web_url": "https://gitlab.com/api/v4/projects/OWNER%2FREPO/iterations?include_ancestors=true&page=1&per_page=30"
  }
]`

	fakeHTTP.RegisterResponder(http.MethodGet, "https://gitlab.com/api/v4/projects/OWNER%2FREPO/iterations?include_ancestors=true&page=1&per_page=30",
		httpmock.NewStringResponse(http.StatusOK, `[
  {
    "id": 53,
    "iid": 13,
    "group_id": 5,
    "title": "Iteration II",
    "description": "Ipsum Lorem ipsum",
    "state": 2,
    "created_at": "2020-01-27T05:07:12.573Z",
    "updated_at": "2020-01-27T05:07:12.573Z",
    "due_date": "2020-02-01",
    "start_date": "2020-02-14",
    "web_url": "https://gitlab.com/api/v4/projects/OWNER%2FREPO/iterations?include_ancestors=true&page=1&per_page=30"
  }
]`))

	output, err := runCommand(t, fakeHTTP, "-F json")
	if err != nil {
		t.Errorf("error running command `iteration list -F json`: %v", err)
	}

	assert.JSONEq(t, expectedBody, output.String())
	assert.Empty(t, output.Stderr())
}
