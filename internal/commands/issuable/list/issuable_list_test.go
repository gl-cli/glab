package list

import (
	"bufio"
	"fmt"
	"net/http"
	"os"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"testing"

	"gitlab.com/gitlab-org/cli/internal/glinstance"
	"gitlab.com/gitlab-org/cli/internal/iostreams"

	"gitlab.com/gitlab-org/cli/internal/commands/issuable"
	"gitlab.com/gitlab-org/cli/internal/testing/cmdtest"

	"github.com/MakeNowJust/heredoc/v2"

	"github.com/stretchr/testify/assert"
	gitlab "gitlab.com/gitlab-org/api/client-go"
	"gitlab.com/gitlab-org/cli/internal/api"
	"gitlab.com/gitlab-org/cli/internal/testing/httpmock"
	"gitlab.com/gitlab-org/cli/test"
)

func runCommand(t *testing.T, command string, rt http.RoundTripper, isTTY bool, cli string, doHyperlinks string) (*test.CmdOut, error) {
	ios, _, stdout, stderr := cmdtest.TestIOStreams(cmdtest.WithTestIOStreamsAsTTY(isTTY), iostreams.WithDisplayHyperLinks(doHyperlinks))
	c := cmdtest.NewTestApiClient(t, &http.Client{Transport: rt}, "", glinstance.DefaultHostname)
	factory := cmdtest.NewTestFactory(ios,
		cmdtest.WithApiClient(c),
		cmdtest.WithGitLabClient(c.Lab()),
	)

	issueType := issuable.TypeIssue
	if command == "incident" {
		issueType = issuable.TypeIncident
	}

	cmd := NewCmdList(factory, nil, issueType)

	return cmdtest.ExecuteCommand(cmd, cli, stdout, stderr)
}

func TestNewCmdList(t *testing.T) {
	ios, _, _, _ := cmdtest.TestIOStreams(cmdtest.WithTestIOStreamsAsTTY(true))

	fakeHTTP := httpmock.New()
	defer fakeHTTP.Verify(t)

	factory := cmdtest.NewTestFactory(ios,
		cmdtest.WithGitLabClient(cmdtest.NewTestApiClient(t, &http.Client{Transport: fakeHTTP}, "", glinstance.DefaultHostname).Lab()),
	)
	t.Run("Issue_NewCmdList", func(t *testing.T) {
		gotOpts := &ListOptions{}
		err := NewCmdList(factory, func(opts *ListOptions) error {
			gotOpts = opts
			return nil
		}, issuable.TypeIssue).Execute()

		assert.Nil(t, err)
		assert.Equal(t, factory.IO(), gotOpts.IO)

		gotBaseRepo, _ := gotOpts.BaseRepo()
		expectedBaseRepo, _ := factory.BaseRepo()
		assert.Equal(t, gotBaseRepo, expectedBaseRepo)
	})
}

func TestIssueList_tty(t *testing.T) {
	// NOTE: we need to force disable colors, otherwise we'd need ANSI sequences in our test output assertions.
	t.Setenv("NO_COLOR", "true")

	fakeHTTP := httpmock.New()
	defer fakeHTTP.Verify(t)

	fakeHTTP.RegisterResponder(http.MethodGet, "/projects/OWNER/REPO/issues",
		httpmock.NewFileResponse(http.StatusOK, "./testdata/issuableList.json"))

	output, err := runCommand(t, "issue", fakeHTTP, true, "", "")
	if err != nil {
		t.Errorf("error running command `issue list`: %v", err)
	}

	out := output.String()
	timeRE := regexp.MustCompile(`\d+ years`)
	out = timeRE.ReplaceAllString(out, "X years")

	assert.Equal(t, heredoc.Doc(`
		Showing 3 open issues in OWNER/REPO that match your search. (Page 1)

		#6	OWNER/REPO/issues/6	Issue one	(foo, bar) 	about X years ago
		#7	OWNER/REPO/issues/7	Issue two	(fooz, baz)	about X years ago
		#8	OWNER/REPO/issues/8	Incident 	(foo, baz) 	about X years ago

	`), out)
	assert.Equal(t, ``, output.Stderr())
}

func TestIssueList_ids(t *testing.T) {
	fakeHTTP := httpmock.New()
	defer fakeHTTP.Verify(t)

	fakeHTTP.RegisterResponder(http.MethodGet, "/projects/OWNER/REPO/issues",
		httpmock.NewFileResponse(http.StatusOK, "./testdata/issuableList.json"))

	output, err := runCommand(t, "issue", fakeHTTP, true, "-F ids", "")
	if err != nil {
		t.Errorf("error running command `issue list -F ids`: %v", err)
	}

	out := output.String()

	assert.Equal(t, "6\n7\n8\n", out)
	assert.Equal(t, ``, output.Stderr())
}

func TestIssueList_urls(t *testing.T) {
	fakeHTTP := httpmock.New()
	defer fakeHTTP.Verify(t)

	fakeHTTP.RegisterResponder(http.MethodGet, "/projects/OWNER/REPO/issues",
		httpmock.NewFileResponse(http.StatusOK, "./testdata/issuableList.json"))

	output, err := runCommand(t, "issue", fakeHTTP, true, "-F urls", "")
	if err != nil {
		t.Errorf("error running command `issue list -F urls`: %v", err)
	}

	out := output.String()

	assert.Equal(t, heredoc.Doc(`
		http://gitlab.com/OWNER/REPO/issues/6
		http://gitlab.com/OWNER/REPO/issues/7
		http://gitlab.com/OWNER/REPO/issues/8
	`), out)
	assert.Equal(t, ``, output.Stderr())
}

func TestIssueList_tty_withFlags(t *testing.T) {
	t.Run("project", func(t *testing.T) {
		fakeHTTP := httpmock.New()
		defer fakeHTTP.Verify(t)

		fakeHTTP.RegisterResponder(http.MethodGet, "/users",
			httpmock.NewStringResponse(http.StatusOK, `[{"id": 100, "username": "someuser"}]`))
		fakeHTTP.RegisterResponder(http.MethodGet, "/projects/OWNER/REPO/issues",
			httpmock.NewStringResponse(http.StatusOK, `[]`))

		output, err := runCommand(t, "issue", fakeHTTP, true, "--opened -P1 -p100 --confidential -a someuser -l bug -m1", "")
		if err != nil {
			t.Errorf("error running command `issue list`: %v", err)
		}

		assert.Equal(t, "", output.Stderr())
		assert.Equal(t, output.String(), `No open issues match your search in OWNER/REPO.


`, output.String())
	})
	t.Run("group", func(t *testing.T) {
		fakeHTTP := httpmock.New()
		defer fakeHTTP.Verify(t)

		fakeHTTP.RegisterResponder(http.MethodGet, "/groups/GROUP/issues",
			httpmock.NewStringResponse(http.StatusOK, `[]`))

		output, err := runCommand(t, "issue", fakeHTTP, true, "--group GROUP", "")
		if err != nil {
			t.Errorf("error running command `issue list`: %v", err)
		}

		assert.Equal(t, "", output.Stderr())
		assert.Equal(t, `No open issues match your search in GROUP.


`, output.String())
	})
}

func TestIssueList_filterByIteration(t *testing.T) {
	fakeHTTP := &httpmock.Mocker{
		MatchURL: httpmock.PathAndQuerystring,
	}
	defer fakeHTTP.Verify(t)

	fakeHTTP.RegisterResponder(http.MethodGet, "/api/v4/projects/OWNER/REPO/issues?in=title%2Cdescription&iteration_id=9&page=1&per_page=30&state=opened",
		httpmock.NewStringResponse(http.StatusOK, `[]`))

	output, err := runCommand(t, "issue", fakeHTTP, true, "--iteration 9", "")
	if err != nil {
		t.Errorf("error running command `issue list`: %v", err)
	}

	assert.Equal(t, "", output.Stderr())
	assert.Equal(t, `No open issues match your search in OWNER/REPO.


`, output.String())
}

func TestIssueList_tty_withIssueType(t *testing.T) {
	// NOTE: we need to force disable colors, otherwise we'd need ANSI sequences in our test output assertions.
	t.Setenv("NO_COLOR", "true")

	fakeHTTP := httpmock.New()
	defer fakeHTTP.Verify(t)

	fakeHTTP.RegisterResponder(http.MethodGet, "/projects/OWNER/REPO/issues",
		httpmock.NewFileResponse(http.StatusOK, "./testdata/incidentList.json"))

	output, err := runCommand(t, "issue", fakeHTTP, true, "--issue-type=incident", "")
	if err != nil {
		t.Errorf("error running command `issue list`: %v", err)
	}

	out := output.String()
	timeRE := regexp.MustCompile(`\d+ years`)
	out = timeRE.ReplaceAllString(out, "X years")

	assert.Equal(t, heredoc.Doc(`
		Showing 1 open incident in OWNER/REPO that match your search. (Page 1)

		#8	OWNER/REPO/issues/8	Incident	(foo, baz)	about X years ago

	`), out)
	assert.Equal(t, ``, output.Stderr())
}

func TestIncidentList_tty_withIssueType(t *testing.T) {
	fakeHTTP := httpmock.New()

	fakeHTTP.RegisterResponder(http.MethodGet, "/projects/OWNER/REPO/issues",
		httpmock.NewFileResponse(http.StatusOK, "./testdata/incidentList.json"))

	output, err := runCommand(t, "incident", fakeHTTP, true, "--issue-type=incident", "")
	if err == nil {
		t.Error("expected an `unknown flag: --issue-type` error, but got nothing")
	}

	assert.Equal(t, ``, output.String())
	assert.Equal(t, ``, output.Stderr())
}

func TestIssueList_tty_mine(t *testing.T) {
	t.Run("mine with all flag and user exists", func(t *testing.T) {
		fakeHTTP := httpmock.New()
		defer fakeHTTP.Verify(t)

		fakeHTTP.RegisterResponder(http.MethodGet, "/projects/OWNER/REPO/issues",
			httpmock.NewStringResponse(http.StatusOK, `[]`))

		fakeHTTP.RegisterResponder(http.MethodGet, "/user",
			httpmock.NewStringResponse(http.StatusOK, `{"username": "john_smith"}`))

		output, err := runCommand(t, "issue", fakeHTTP, true, "--mine -A", "")
		if err != nil {
			t.Errorf("error running command `issue list`: %v", err)
		}

		assert.Equal(t, "", output.Stderr(), "")
		assert.Equal(t, `No issues match your search in OWNER/REPO.


`, output.String())
	})
	t.Run("user does not exists", func(t *testing.T) {
		fakeHTTP := httpmock.New()
		defer fakeHTTP.Verify(t)

		fakeHTTP.RegisterResponder(http.MethodGet, "/user",
			httpmock.NewStringResponse(http.StatusNotFound, `{message: 404 Not found}`))

		output, err := runCommand(t, "issue", fakeHTTP, true, "--mine -A", "")
		assert.NotNil(t, err)

		assert.Equal(t, "", output.Stderr())
		assert.Equal(t, "", output.String())
	})
}

func makeHyperlink(linkText, targetURL string) string {
	return fmt.Sprintf("\x1b]8;;%s\x1b\\%s\x1b]8;;\x1b\\", targetURL, linkText)
}

func TestIssueList_hyperlinks(t *testing.T) {
	// NOTE: we need to force disable colors, otherwise we'd need ANSI sequences in our test output assertions.
	t.Setenv("NO_COLOR", "true")

	noHyperlinkCells := [][]string{
		{"#6", "OWNER/REPO/issues/6", "Issue one", "(foo, bar)", "about X years ago"},
		{"#7", "OWNER/REPO/issues/7", "Issue two", "(fooz, baz)", "about X years ago"},
		{"#8", "OWNER/REPO/issues/8", "Incident", "(foo, baz)", "about X years ago"},
	}

	hyperlinkCells := [][]string{
		{makeHyperlink("#6", "http://gitlab.com/OWNER/REPO/issues/6"), "OWNER/REPO/issues/6", "Issue one", "(foo, bar)", "about X years ago"},
		{makeHyperlink("#7", "http://gitlab.com/OWNER/REPO/issues/7"), "OWNER/REPO/issues/7", "Issue two", "(fooz, baz)", "about X years ago"},
		{makeHyperlink("#8", "http://gitlab.com/OWNER/REPO/issues/8"), "OWNER/REPO/issues/8", "Incident", "(foo, baz)", "about X years ago"},
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

			fakeHTTP.RegisterResponder(http.MethodGet, "/projects/OWNER/REPO/issues",
				httpmock.NewFileResponse(http.StatusOK, "./testdata/issuableList.json"))

			doHyperlinks := "never"
			if test.forceHyperlinksEnv == "1" {
				doHyperlinks = "always"
			} else if test.displayHyperlinksConfig == "true" {
				doHyperlinks = "auto"
			}

			output, err := runCommand(t, "issue", fakeHTTP, test.isTTY, "", doHyperlinks)
			if err != nil {
				t.Errorf("error running command `issue list`: %v", err)
			}

			out := output.String()
			timeRE := regexp.MustCompile(`\d+ years`)
			out = timeRE.ReplaceAllString(out, "X years")

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

func TestIssueListJSON(t *testing.T) {
	fakeHTTP := httpmock.New()
	defer fakeHTTP.Verify(t)

	fakeHTTP.RegisterResponder(http.MethodGet, "/projects/OWNER/REPO/issues",
		httpmock.NewFileResponse(http.StatusOK, "./testdata/issueListFull.json"))

	output, err := runCommand(t, "issue", fakeHTTP, true, "--output json", "")
	if err != nil {
		t.Errorf("error running command `issue list -F json`: %v", err)
	}

	if err != nil {
		panic(err)
	}

	b, err := os.ReadFile("./testdata/issueListFull.json")
	if err != nil {
		fmt.Print(err)
	}

	expectedOut := string(b)

	assert.JSONEq(t, expectedOut, output.String())
	assert.Empty(t, output.Stderr())
}

func TestIssueListMutualOutputFlags(t *testing.T) {
	_, err := runCommand(t, "issue", nil, true, "--output json --output-format ids", "")

	assert.NotNil(t, err)
	assert.EqualError(t, err, "if any flags in the group [output output-format] are set none of the others can be; [output output-format] were all set")
}

func TestIssueList_epicIssues(t *testing.T) {
	testdata := []*gitlab.Issue{
		{
			IID:   1,
			State: "opened",
			Assignees: []*gitlab.IssueAssignee{
				{ID: 101},
			},
			Author: &gitlab.IssueAuthor{ID: 102},
			Labels: gitlab.Labels{"label::one"},
			Milestone: &gitlab.Milestone{
				Title: "Milestone one",
			},
			Title: "This is issue one",
			Iteration: &gitlab.GroupIteration{
				ID: 103,
			},
			Confidential: false,
		},
		{
			IID:   2,
			State: "closed",
			Assignees: []*gitlab.IssueAssignee{
				{ID: 102},
			},
			Author: &gitlab.IssueAuthor{ID: 202},
			Labels: gitlab.Labels{"label::two"},
			Milestone: &gitlab.Milestone{
				Title: "Milestone two",
			},
			Title: "That is issue two",
			Iteration: &gitlab.GroupIteration{
				ID: 203,
			},
			Confidential: true,
		},
	}

	tests := []struct {
		name        string
		commandLine string
		expectedURL string
		user        *gitlab.User
		wantIDs     []int
		wantErr     string
	}{
		{
			name:        "group flag",
			commandLine: `--group testGroupID --epic 42`,
			wantIDs:     []int{1},
		},
		{
			name:        "repo flag",
			commandLine: `--repo testGroupID/repo --epic 42`,
			wantIDs:     []int{1},
		},
		{
			name:        "all flag",
			commandLine: `--group testGroupID --epic 42 --all`,
			wantIDs:     []int{1, 2},
		},
		{
			name:        "closed flag",
			commandLine: `--group testGroupID --epic 42 --closed`,
			wantIDs:     []int{2},
		},
		{
			name: "assignee flag",
			user: &gitlab.User{
				ID:       101,
				Username: "one-oh-one",
			},
			commandLine: `--group testGroupID --epic 42 --all --assignee one-oh-one`,
			wantIDs:     []int{1},
		},
		{
			name: "not-assignee flag",
			user: &gitlab.User{
				ID:       101,
				Username: "one-oh-one",
			},
			commandLine: `--group testGroupID --epic 42 --all --not-assignee one-oh-one`,
			wantIDs:     []int{2},
		},
		{
			name: "author flag",
			user: &gitlab.User{
				ID:       102,
				Username: "one-oh-two",
			},
			commandLine: `--group testGroupID --epic 42 --all --author one-oh-two`,
			wantIDs:     []int{1},
		},
		{
			name: "not-author flag",
			user: &gitlab.User{
				ID:       102,
				Username: "one-oh-two",
			},
			commandLine: `--group testGroupID --epic 42 --all --not-author one-oh-two`,
			wantIDs:     []int{2},
		},
		{
			name:        "label flag",
			commandLine: `--group testGroupID --epic 42 --all --label 'label::one'`,
			wantIDs:     []int{1},
		},
		{
			name:        "not-label flag",
			commandLine: `--group testGroupID --epic 42 --all --not-label 'label::one'`,
			wantIDs:     []int{2},
		},
		{
			name:        "milestone flag",
			commandLine: `--group testGroupID --epic 42 --all --milestone 'milestone one'`,
			wantIDs:     []int{1},
		},
		{
			name:        "search flag",
			commandLine: `--group testGroupID --epic 42 --all --search 'iSsUe OnE'`,
			wantIDs:     []int{1},
		},
		{
			name:        "iteration flag",
			commandLine: `--group testGroupID --epic 42 --all --iteration 103`,
			wantIDs:     []int{1},
		},
		{
			name:        "confidential flag",
			commandLine: `--group testGroupID --epic 42 --all --confidential`,
			wantIDs:     []int{2},
		},
		{
			name:        "page flag",
			commandLine: `--group testGroupID --epic 42 --all --page=2`,
			wantErr:     "the --page flag",
		},
		{
			name:        "per-page flag",
			commandLine: `--group testGroupID --epic 42 --all --per-page=9999`,
			// per-page is clamped to the max supported per_page value
			expectedURL: fmt.Sprintf(`/api/v4/groups/testGroupID/epics/42/issues?page=1&per_page=%d`, api.MaxPerPage),
			wantIDs:     []int{1, 2},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fakeHTTP := &httpmock.Mocker{
				MatchURL: httpmock.PathAndQuerystring,
			}
			defer fakeHTTP.Verify(t)

			if tt.user != nil {
				fakeHTTP.RegisterResponder(http.MethodGet, "/api/v4/users?per_page=30&username="+tt.user.Username,
					httpmock.NewJSONResponse(http.StatusOK, []*gitlab.User{tt.user}))
			}

			if tt.wantErr == "" {
				expectedURL := tt.expectedURL
				if expectedURL == "" {
					expectedURL = `/api/v4/groups/testGroupID/epics/42/issues?page=1&per_page=30`
				}
				fakeHTTP.RegisterResponder(http.MethodGet, expectedURL, httpmock.NewJSONResponse(http.StatusOK, testdata))
			}

			output, err := runCommand(t, "issue", fakeHTTP, true, tt.commandLine+` --output-format ids`, "")
			if tt.wantErr != "" {
				assert.ErrorContains(t, err, tt.wantErr)
			}
			if err != nil {
				return
			}

			assert.Equal(t, "", output.Stderr())

			gotIDs, err := strToIntSlice(output.String())
			if err != nil {
				t.Fatalf("command %q: unexpected output:\n%s", tt.commandLine, output.String())
			}

			assert.Equal(t, tt.wantIDs, gotIDs)
		})
	}
}

func strToIntSlice(s string) ([]int, error) {
	var ret []int

	scanner := bufio.NewScanner(strings.NewReader(s))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		i, err := strconv.Atoi(line)
		if err != nil {
			return nil, err
		}

		ret = append(ret, i)
	}

	slices.Sort(ret)

	return ret, nil
}
