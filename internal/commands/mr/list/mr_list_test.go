package list

import (
	"fmt"
	"net/http"
	"os"
	"strings"
	"testing"

	"gitlab.com/gitlab-org/cli/internal/glinstance"
	"gitlab.com/gitlab-org/cli/internal/iostreams"

	"gitlab.com/gitlab-org/cli/internal/testing/cmdtest"

	"github.com/MakeNowJust/heredoc/v2"

	"github.com/stretchr/testify/assert"
	"gitlab.com/gitlab-org/cli/internal/config"
	"gitlab.com/gitlab-org/cli/internal/testing/httpmock"
	"gitlab.com/gitlab-org/cli/test"
)

func runCommand(t *testing.T, rt http.RoundTripper, isTTY bool, cli string, doHyperlinks string) (*test.CmdOut, error) {
	ios, _, stdout, stderr := cmdtest.TestIOStreams(cmdtest.WithTestIOStreamsAsTTY(isTTY), iostreams.WithDisplayHyperLinks(doHyperlinks))
	factory := cmdtest.NewTestFactory(ios,
		cmdtest.WithApiClient(cmdtest.NewTestApiClient(t, &http.Client{Transport: rt}, "", glinstance.DefaultHostname)),
	)

	cmd := NewCmdList(factory, nil)

	return cmdtest.ExecuteCommand(cmd, cli, stdout, stderr)
}

func TestNewCmdList(t *testing.T) {
	ios, _, _, _ := cmdtest.TestIOStreams(cmdtest.WithTestIOStreamsAsTTY(true))

	fakeHTTP := httpmock.New()
	defer fakeHTTP.Verify(t)

	factory := cmdtest.NewTestFactory(ios,
		cmdtest.WithApiClient(cmdtest.NewTestApiClient(t, &http.Client{Transport: fakeHTTP}, "", "")),
		cmdtest.WithConfig(config.NewBlankConfig()),
		cmdtest.WithBaseRepo("OWNER", "REPO"),
	)
	t.Run("MergeRequest_NewCmdList", func(t *testing.T) {
		gotOpts := &options{}
		err := NewCmdList(factory, func(opts *options) error {
			gotOpts = opts
			return nil
		}).Execute()

		assert.Nil(t, err)
		assert.Equal(t, factory.IO(), gotOpts.io)

		gotBaseRepo, _ := gotOpts.baseRepo()
		expectedBaseRepo, _ := factory.BaseRepo()
		assert.Equal(t, gotBaseRepo, expectedBaseRepo)
	})
}

func TestMergeRequestList_tty(t *testing.T) {
	fakeHTTP := httpmock.New()
	defer fakeHTTP.Verify(t)

	fakeHTTP.RegisterResponder(http.MethodGet, "/projects/OWNER/REPO/merge_requests",
		httpmock.NewStringResponse(http.StatusOK, `
[
  {
    "state": "opened",
    "description": "a description here",
    "project_id": 1,
    "updated_at": "2016-01-04T15:31:51.081Z",
    "id": 76,
    "title": "MergeRequest one",
    "created_at": "2016-01-04T15:31:51.081Z",
    "iid": 6,
    "labels": ["foo", "bar"],
	"target_branch": "master",
    "source_branch": "test1",
    "web_url": "http://gitlab.com/OWNER/REPO/merge_requests/6",
    "references": {
      "full": "OWNER/REPO/merge_requests/6",
      "relative": "#6",
      "short": "#6"
    }
  },
  {
    "state": "opened",
    "description": "description two here",
    "project_id": 1,
    "updated_at": "2016-01-04T15:31:51.081Z",
    "id": 77,
    "title": "MergeRequest two",
    "created_at": "2016-01-04T15:31:51.081Z",
    "iid": 7,
	"target_branch": "master",
    "source_branch": "test2",
    "labels": ["fooz", "baz"],
    "web_url": "http://gitlab.com/OWNER/REPO/merge_requests/7",
    "references": {
      "full": "OWNER/REPO/merge_requests/7",
      "relative": "#7",
      "short": "#7"
    }
  }
]
`))

	// NOTE: we need to force disable colors, otherwise we'd need ANSI sequences in our test output assertions.
	t.Setenv("NO_COLOR", "true")

	output, err := runCommand(t, fakeHTTP, true, "", "")
	if err != nil {
		t.Errorf("error running command `issue list`: %v", err)
	}

	assert.Equal(t, heredoc.Doc(`
		Showing 2 open merge requests on OWNER/REPO. (Page 1)

		!6	OWNER/REPO/merge_requests/6	MergeRequest one	(master) ← (test1)
		!7	OWNER/REPO/merge_requests/7	MergeRequest two	(master) ← (test2)

	`), output.String())
	assert.Equal(t, ``, output.Stderr())
}

func TestMergeRequestList_tty_withFlags(t *testing.T) {
	// NOTE: we need to force disable colors, otherwise we'd need ANSI sequences in our test output assertions.
	t.Setenv("NO_COLOR", "true")

	t.Run("repo", func(t *testing.T) {
		fakeHTTP := httpmock.New()
		defer fakeHTTP.Verify(t)

		fakeHTTP.RegisterResponder(http.MethodGet, "/projects/OWNER/REPO/merge_requests",
			httpmock.NewStringResponse(http.StatusOK, `[]`))

		fakeHTTP.RegisterResponder(http.MethodGet, "/users",
			httpmock.NewStringResponse(http.StatusOK, `[{"id": 1, "iid": 1, "username": "john_smith"}]`))

		output, err := runCommand(t, fakeHTTP, true, "--opened -P1 -p100 -a someuser -l bug -m1", "")
		if err != nil {
			t.Errorf("error running command `issue list`: %v", err)
		}

		assert.Equal(t, "", output.Stderr())
		assert.Equal(t, `No open merge requests match your search in OWNER/REPO.


`, output.String())
	})
	t.Run("group", func(t *testing.T) {
		fakeHTTP := httpmock.New()
		defer fakeHTTP.Verify(t)

		fakeHTTP.RegisterResponder(http.MethodGet, "/groups/GROUP/merge_requests",
			httpmock.NewStringResponse(http.StatusOK, `[]`))

		output, err := runCommand(t, fakeHTTP, true, "--group GROUP", "")
		if err != nil {
			t.Errorf("error running command `mr list`: %v", err)
		}

		assert.Equal(t, "", output.Stderr())
		assert.Equal(t, `No open merge requests available on GROUP.

`, output.String())
	})

	t.Run("draft", func(t *testing.T) {
		fakeHTTP := &httpmock.Mocker{
			MatchURL: httpmock.PathAndQuerystring,
		}
		defer fakeHTTP.Verify(t)

		fakeHTTP.RegisterResponder(http.MethodGet, "/api/v4/projects/OWNER%2FREPO/merge_requests?page=1&per_page=30&state=opened&wip=yes",
			httpmock.NewStringResponse(http.StatusOK, `[
				{
					"state": "opened",
					"description": "a description here",
					"project_id": 1,
					"updated_at": "2016-01-04T15:31:51.081Z",
					"id": 76,
					"title": "MergeRequest one",
					"created_at": "2016-01-04T15:31:51.081Z",
					"iid": 6,
					"draft": true,
					"labels": ["foo", "bar"],
					"target_branch": "master",
					"source_branch": "test1",
					"web_url": "http://gitlab.com/OWNER/REPO/merge_requests/6",
					"references": {
					"full": "OWNER/REPO/merge_requests/6",
					"relative": "#6",
					"short": "#6"
					}
				},
				{
					"state": "opened",
					"description": "description two here",
					"project_id": 1,
					"updated_at": "2016-01-04T15:31:51.081Z",
					"id": 77,
					"title": "MergeRequest two",
					"created_at": "2016-01-04T15:31:51.081Z",
					"iid": 7,
					"draft": true,
					"target_branch": "master",
					"source_branch": "test2",
					"labels": ["fooz", "baz"],
					"web_url": "http://gitlab.com/OWNER/REPO/merge_requests/7",
					"references": {
					  "full": "OWNER/REPO/merge_requests/7",
					  "relative": "#7",
					  "short": "#7"
					}
				}
			]`))

		output, err := runCommand(t, fakeHTTP, true, "--draft", "")
		if err != nil {
			t.Errorf("error running command `mr list`: %v", err)
		}

		assert.Equal(t, output.Stderr(), "")
		assert.Equal(t, heredoc.Doc(`
		Showing 2 open merge requests in OWNER/REPO that match your search. (Page 1)

		!6	OWNER/REPO/merge_requests/6	MergeRequest one	(master) ← (test1)
		!7	OWNER/REPO/merge_requests/7	MergeRequest two	(master) ← (test2)

	`), output.String())
	})

	t.Run("not draft", func(t *testing.T) {
		fakeHTTP := &httpmock.Mocker{
			MatchURL: httpmock.PathAndQuerystring,
		}
		defer fakeHTTP.Verify(t)

		fakeHTTP.RegisterResponder(http.MethodGet, "/api/v4/projects/OWNER/REPO/merge_requests?page=1&per_page=30&state=opened&wip=no",
			httpmock.NewStringResponse(http.StatusOK, `[
			]`))

		output, err := runCommand(t, fakeHTTP, true, "--not-draft", "")
		if err != nil {
			t.Errorf("error running command `mr list`: %v", err)
		}

		assert.Equal(t, output.Stderr(), "")
		assert.Equal(t, "No open merge requests match your search in OWNER/REPO.\n\n\n", output.String())
	})
}

func makeHyperlink(linkText, targetURL string) string {
	return fmt.Sprintf("\x1b]8;;%s\x1b\\%s\x1b]8;;\x1b\\", targetURL, linkText)
}

func TestMergeRequestList_hyperlinks(t *testing.T) {
	// NOTE: we need to force disable colors, otherwise we'd need ANSI sequences in our test output assertions.
	t.Setenv("NO_COLOR", "true")

	noHyperlinkCells := [][]string{
		{"!6", "OWNER/REPO/merge_requests/6", "MergeRequest one", "(master) ← (test1)"},
		{"!7", "OWNER/REPO/merge_requests/7", "MergeRequest two", "(master) ← (test2)"},
	}

	hyperlinkCells := [][]string{
		{makeHyperlink("!6", "http://gitlab.com/OWNER/REPO/merge_requests/6"), "OWNER/REPO/merge_requests/6", "MergeRequest one", "(master) ← (test1)"},
		{makeHyperlink("!7", "http://gitlab.com/OWNER/REPO/merge_requests/7"), "OWNER/REPO/merge_requests/7", "MergeRequest two", "(master) ← (test2)"},
	}

	type hyperlinkTest struct {
		forceHyperlinksEnv      string
		displayHyperlinksConfig string
		isTTY                   bool

		expectedCells [][]string
	}

	tests := []hyperlinkTest{
		// FORCE_HYPERLINKS causes hyperlinks to be output, whether or not we're talking to a TTY
		{forceHyperlinksEnv: "1", isTTY: true, expectedCells: hyperlinkCells},
		{forceHyperlinksEnv: "1", isTTY: false, expectedCells: hyperlinkCells},

		// empty/missing display_hyperlinks in config defaults to *not* outputting hyperlinks
		{displayHyperlinksConfig: "", isTTY: true, expectedCells: noHyperlinkCells},
		{displayHyperlinksConfig: "", isTTY: false, expectedCells: noHyperlinkCells},

		// display_hyperlinks: false in config prevents outputting hyperlinks
		{displayHyperlinksConfig: "false", isTTY: true, expectedCells: noHyperlinkCells},
		{displayHyperlinksConfig: "false", isTTY: false, expectedCells: noHyperlinkCells},

		// display_hyperlinks: true in config only outputs hyperlinks if we're talking to a TTY
		{displayHyperlinksConfig: "true", isTTY: true, expectedCells: hyperlinkCells},
		{displayHyperlinksConfig: "true", isTTY: false, expectedCells: noHyperlinkCells},
	}

	for _, test := range tests {
		t.Run("", func(t *testing.T) {
			fakeHTTP := httpmock.New()
			defer fakeHTTP.Verify(t)

			fakeHTTP.RegisterResponder(http.MethodGet, "/projects/OWNER/REPO/merge_requests",
				httpmock.NewStringResponse(http.StatusOK, `
[
  {
    "state": "opened",
    "description": "a description here",
    "project_id": 1,
    "updated_at": "2016-01-04T15:31:51.081Z",
    "id": 76,
    "title": "MergeRequest one",
    "created_at": "2016-01-04T15:31:51.081Z",
    "iid": 6,
    "labels": ["foo", "bar"],
	"target_branch": "master",
    "source_branch": "test1",
    "web_url": "http://gitlab.com/OWNER/REPO/merge_requests/6",
    "references": {
      "full": "OWNER/REPO/merge_requests/6",
      "relative": "#6",
      "short": "#6"
    }
  },
  {
    "state": "opened",
    "description": "description two here",
    "project_id": 1,
    "updated_at": "2016-01-04T15:31:51.081Z",
    "id": 77,
    "title": "MergeRequest two",
    "created_at": "2016-01-04T15:31:51.081Z",
    "iid": 7,
	"target_branch": "master",
    "source_branch": "test2",
    "labels": ["fooz", "baz"],
    "web_url": "http://gitlab.com/OWNER/REPO/merge_requests/7",
    "references": {
      "full": "OWNER/REPO/merge_requests/7",
      "relative": "#7",
      "short": "#7"
    }
  }
]
`))

			doHyperlinks := "never"
			if test.forceHyperlinksEnv == "1" {
				doHyperlinks = "always"
			} else if test.displayHyperlinksConfig == "true" {
				doHyperlinks = "auto"
			}

			output, err := runCommand(t, fakeHTTP, test.isTTY, "", doHyperlinks)
			if err != nil {
				t.Errorf("error running command `mr list`: %v", err)
			}

			out := output.String()

			lines := strings.Split(strings.TrimRight(out, "\n"), "\n")

			// first two lines have the header and some separating whitespace, so skip those
			for lineNum, line := range lines[2:] {
				gotCells := strings.Split(line, "\t")
				expectedCells := test.expectedCells[lineNum]

				assert.Equal(t, len(expectedCells), len(gotCells))

				for cellNum, gotCell := range gotCells {
					expectedCell := expectedCells[cellNum]

					assert.Equal(t, expectedCell, strings.Trim(gotCell, " "))
				}
			}
		})
	}
}

func TestMergeRequestList_labels(t *testing.T) {
	// NOTE: we need to force disable colors, otherwise we'd need ANSI sequences in our test output assertions.
	t.Setenv("NO_COLOR", "true")

	type labelTest struct {
		cli           string
		expectedQuery string
	}

	tests := []labelTest{
		{cli: "--label foo", expectedQuery: "labels=foo&page=1&per_page=30&state=opened"},
		{cli: "--not-label fooz", expectedQuery: "not%5Blabels%5D=fooz&page=1&per_page=30&state=opened"},
	}

	for _, test := range tests {
		t.Run("", func(t *testing.T) {
			fakeHTTP := &httpmock.Mocker{
				MatchURL: httpmock.PathAndQuerystring,
			}
			defer fakeHTTP.Verify(t)

			path := fmt.Sprintf("/api/v4/projects/OWNER/REPO/merge_requests?%s", test.expectedQuery)
			fakeHTTP.RegisterResponder(http.MethodGet, path,
				httpmock.NewStringResponse(http.StatusOK, `
		[
		  {
			"state": "opened",
			"description": "a description here",
			"project_id": 1,
			"updated_at": "2016-01-04T15:31:51.081Z",
			"id": 76,
			"title": "MergeRequest one",
			"created_at": "2016-01-04T15:31:51.081Z",
			"iid": 6,
			"labels": ["foo", "bar"],
			"target_branch": "master",
			"source_branch": "test1",
			"web_url": "http://gitlab.com/OWNER/REPO/merge_requests/6",
			"references": {
			  "full": "OWNER/REPO/merge_requests/6",
			  "relative": "#6",
			  "short": "#6"
			}
		  }
		]
		`))
			output, err := runCommand(t, fakeHTTP, true, test.cli, "")
			if err != nil {
				t.Errorf("error running command `issue list %s`: %v", test.cli, err)
			}

			assert.Contains(t, output.String(), "!6	OWNER/REPO/merge_requests/6")
			assert.Empty(t, output.Stderr())
		})
	}
}

func TestMrListJSON(t *testing.T) {
	fakeHTTP := httpmock.New()
	fakeHTTP.MatchURL = httpmock.PathAndQuerystring
	defer fakeHTTP.Verify(t)

	fakeHTTP.RegisterResponder(http.MethodGet, "/api/v4/projects/OWNER/REPO/merge_requests?page=1&per_page=30&state=opened",
		httpmock.NewFileResponse(http.StatusOK, "./testdata/mrList.json"))

	output, err := runCommand(t, fakeHTTP, true, "-F json", "")
	if err != nil {
		t.Errorf("error running command `mr list -F json`: %v", err)
	}

	if err != nil {
		panic(err)
	}

	b, err := os.ReadFile("./testdata/mrList.json")
	if err != nil {
		fmt.Print(err)
	}

	expectedOut := string(b)

	assert.JSONEq(t, expectedOut, output.String())
	assert.Empty(t, output.Stderr())
}

func TestMergeRequestList_GroupAndReviewer(t *testing.T) {
	// NOTE: we need to force disable colors, otherwise we'd need ANSI sequences in our test output assertions.
	t.Setenv("NO_COLOR", "true")

	fakeHTTP := httpmock.New()
	defer fakeHTTP.Verify(t)

	fakeHTTP.RegisterResponder(http.MethodGet, "/api/v4/user",
		httpmock.NewStringResponse(http.StatusOK, `{"id": 1, "username": "me"}`))

	fakeHTTP.RegisterResponder(http.MethodGet, "/api/v4/groups/GROUP/merge_requests",
		httpmock.NewStringResponse(http.StatusOK, `
[
  {
    "state" : "opened",
    "description" : "a description here",
    "project_id" : 1,
    "updated_at" : "2016-01-04T15:31:51.081Z",
    "id" : 76,
    "title" : "MergeRequest one",
    "created_at" : "2016-01-04T15:31:51.081Z",
    "iid" : 6,
    "labels" : ["foo", "bar"],
	"target_branch": "master",
    "source_branch": "test1",
    "web_url": "http://gitlab.com/OWNER/REPO/merge_requests/6",
    "references": {
      "full": "OWNER/REPO/merge_requests/6",
      "relative": "#6",
      "short": "#6"
    }
  }
]
`))

	output, err := runCommand(t, fakeHTTP, true, "--group GROUP --reviewer @me", "")
	if err != nil {
		t.Errorf("error running command `mr list`: %v", err)
	}

	assert.Equal(t, heredoc.Doc(`
		Showing 1 open merge request on GROUP. (Page 1)

		!6	OWNER/REPO/merge_requests/6	MergeRequest one	(master) ← (test1)

	`), output.String())
	assert.Equal(t, ``, output.Stderr())

	lastRequest := fakeHTTP.Requests[len(fakeHTTP.Requests)-1]
	assert.Contains(t, lastRequest.URL.RawQuery, "reviewer_id=1")
}

func TestMergeRequestList_GroupAndAssignee(t *testing.T) {
	// NOTE: we need to force disable colors, otherwise we'd need ANSI sequences in our test output assertions.
	t.Setenv("NO_COLOR", "true")

	fakeHTTP := httpmock.New()
	defer fakeHTTP.Verify(t)

	// Add stub for user lookup
	fakeHTTP.RegisterResponder(http.MethodGet, "/api/v4/user",
		httpmock.NewStringResponse(http.StatusOK, `{"id": 1, "username": "me"}`))

	fakeHTTP.RegisterResponder(http.MethodGet, "/api/v4/groups/GROUP/merge_requests",
		httpmock.NewStringResponse(http.StatusOK, `
[
  {
    "state" : "opened",
    "description" : "a description here",
    "project_id" : 1,
    "updated_at" : "2016-01-04T15:31:51.081Z",
    "id" : 76,
    "title" : "MergeRequest one",
    "created_at" : "2016-01-04T15:31:51.081Z",
    "iid" : 6,
    "labels" : ["foo", "bar"],
	"target_branch": "master",
    "source_branch": "test1",
    "web_url": "http://gitlab.com/OWNER/REPO/merge_requests/6",
    "references": {
      "full": "OWNER/REPO/merge_requests/6",
      "relative": "#6",
      "short": "#6"
    }
  }
]
`))

	output, err := runCommand(t, fakeHTTP, true, "--group GROUP --assignee @me", "")
	if err != nil {
		t.Errorf("error running command `mr list`: %v", err)
	}

	assert.Equal(t, heredoc.Doc(`
		Showing 1 open merge request on GROUP. (Page 1)

		!6	OWNER/REPO/merge_requests/6	MergeRequest one	(master) ← (test1)

	`), output.String())
	assert.Equal(t, ``, output.Stderr())

	lastRequest := fakeHTTP.Requests[len(fakeHTTP.Requests)-1]
	assert.Contains(t, lastRequest.URL.RawQuery, "assignee_id=1")
}

func TestMergeRequestList_GroupWithAssigneeAndReviewer(t *testing.T) {
	// NOTE: we need to force disable colors, otherwise we'd need ANSI sequences in our test output assertions.
	t.Setenv("NO_COLOR", "true")

	fakeHTTP := httpmock.New()
	defer fakeHTTP.Verify(t)
	fakeHTTP.MatchURL = httpmock.PathAndQuerystring

	fakeHTTP.RegisterResponder(http.MethodGet, "/api/v4/users?per_page=30&username=some.user",
		httpmock.NewStringResponse(http.StatusOK, `[{"id": 2, "username": "some.user"}]`))

	fakeHTTP.RegisterResponder(http.MethodGet, "/api/v4/users?per_page=30&username=other.user",
		httpmock.NewStringResponse(http.StatusOK, `[{"id": 1, "username": "other.user"}]`))

	fakeHTTP.RegisterResponder(http.MethodGet, "/api/v4/groups/GROUP/merge_requests?assignee_id=1&page=1&per_page=30&state=opened",
		httpmock.NewStringResponse(http.StatusOK, `
[
  {
    "state": "opened",
    "description": "a description here",
    "project_id": 1,
    "updated_at": "2016-01-04T15:31:51.081Z",
    "id": 76,
    "title": "MergeRequest one",
    "created_at": "2016-01-04T15:31:51.081Z",
    "iid": 6,
    "labels": ["foo", "bar"],
    "target_branch": "master",
    "source_branch": "test1",
    "web_url": "http://gitlab.com/OWNER/REPO/merge_requests/6",
    "references": {
      "full": "OWNER/REPO/merge_requests/6",
      "relative": "#6",
      "short": "#6"
    }
  }
]
`))

	fakeHTTP.RegisterResponder(http.MethodGet, "/api/v4/groups/GROUP/merge_requests?page=1&per_page=30&reviewer_id=2&state=opened",
		httpmock.NewStringResponse(http.StatusOK, `
[
  {
    "state": "opened",
    "description": "a description here",
    "project_id": 2,
    "updated_at": "2024-01-04T15:31:51.081Z",
    "id": 77,
    "title": "MergeRequest one",
    "created_at": "2024-01-04T15:31:51.081Z",
    "iid": 7,
    "labels": ["baz", "bar"],
    "target_branch": "master",
    "source_branch": "test2",
    "web_url": "http://gitlab.com/OWNER/REPO/merge_requests/7",
    "references": {
      "full": "OWNER/REPO/merge_requests/7",
      "relative": "#7",
      "short": "#7"
    }
  }
]
`))

	output, err := runCommand(t, fakeHTTP, true, "--group GROUP --reviewer=some.user --assignee=other.user", "")
	if err != nil {
		t.Errorf("error running command `mr list`: %v", err)
	}

	assert.Equal(t, heredoc.Doc(`
		Showing 2 open merge requests on GROUP. (Page 1)

		!7	OWNER/REPO/merge_requests/7	MergeRequest one	(master) ← (test2)
		!6	OWNER/REPO/merge_requests/6	MergeRequest one	(master) ← (test1)

	`), output.String())
	assert.Equal(t, ``, output.Stderr())

	requests := fakeHTTP.Requests
	// 2 for users lookup, 2 for merge requests (assignee and reviewer)
	assert.Len(t, requests, 4)
}

func TestMergeRequestList_SortAndOrderBy(t *testing.T) {
	// NOTE: we need to force disable colors, otherwise we'd need ANSI sequences in our test output assertions.
	t.Setenv("NO_COLOR", "true")

	fakeHTTP := &httpmock.Mocker{
		MatchURL: httpmock.PathAndQuerystring,
	}
	defer fakeHTTP.Verify(t)

	fakeHTTP.RegisterResponder(http.MethodGet, "/api/v4/projects/OWNER/REPO/merge_requests?order_by=created_at&page=1&per_page=30&sort=desc&state=opened",
		httpmock.NewStringResponse(http.StatusOK, `[
				{
					"state": "opened",
					"description": "a description here",
					"project_id": 1,
					"updated_at": "2016-01-04T16:00:00.081Z",
					"id": 76,
					"title": "MergeRequest one",
					"created_at": "2016-01-04T16:00:00.081Z",
					"iid": 6,
					"draft": true,
					"labels": ["foo", "bar"],
					"target_branch": "master",
					"source_branch": "test1",
					"web_url": "http://gitlab.com/OWNER/REPO/merge_requests/6",
					"references": {
					"full": "OWNER/REPO/merge_requests/6",
					"relative": "#6",
					"short": "#6"
					}
				},
				{
					"state": "opened",
					"description": "description two here",
					"project_id": 1,
					"updated_at": "2016-01-04T15:00:00.081Z",
					"id": 77,
					"title": "MergeRequest two",
					"created_at": "2016-01-04T15:00:00.081Z",
					"iid": 7,
					"draft": true,
					"target_branch": "master",
					"source_branch": "test2",
					"labels": ["fooz", "baz"],
					"web_url": "http://gitlab.com/OWNER/REPO/merge_requests/7",
					"references": {
					  "full": "OWNER/REPO/merge_requests/7",
					  "relative": "#7",
					  "short": "#7"
					}
				}
	]`))

	output, err := runCommand(t, fakeHTTP, true, "--order created_at --sort desc", "")
	if err != nil {
		t.Errorf("error running command `mr list`: %v", err)
	}

	assert.Equal(t, output.Stderr(), "")
	assert.Equal(t, heredoc.Doc(`
	Showing 2 open merge requests in OWNER/REPO that match your search. (Page 1)

	!6	OWNER/REPO/merge_requests/6	MergeRequest one	(master) ← (test1)
	!7	OWNER/REPO/merge_requests/7	MergeRequest two	(master) ← (test2)

	`), output.String())
}
