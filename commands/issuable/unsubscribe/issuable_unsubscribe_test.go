package unsubscribe

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/MakeNowJust/heredoc/v2"
	"github.com/google/shlex"
	"github.com/stretchr/testify/require"
	"github.com/xanzy/go-gitlab"
	"gitlab.com/gitlab-org/cli/api"
	"gitlab.com/gitlab-org/cli/commands/cmdutils"
	"gitlab.com/gitlab-org/cli/commands/issuable"
	"gitlab.com/gitlab-org/cli/internal/glrepo"
	"gitlab.com/gitlab-org/cli/pkg/httpmock"
	"gitlab.com/gitlab-org/cli/pkg/iostreams"
	"gitlab.com/gitlab-org/cli/test"
)

func runCommand(rt http.RoundTripper, issuableID string, issueType issuable.IssueType) (*test.CmdOut, error) {
	ios, _, stdout, stderr := iostreams.Test()
	ios.IsaTTY = true
	ios.IsErrTTY = true

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
	}

	_, _ = factory.HttpClient()

	cmd := NewCmdUnsubscribe(factory, issueType)

	argv, err := shlex.Split(issuableID)
	if err != nil {
		return nil, err
	}
	cmd.SetArgs(argv)

	_, err = cmd.ExecuteC()
	return &test.CmdOut{
		OutBuf: stdout,
		ErrBuf: stderr,
	}, err
}

func mockIssuableGet(fakeHTTP *httpmock.Mocker, id int, issueType string, subscribed bool) {
	fakeHTTP.RegisterResponder(http.MethodGet, fmt.Sprintf("/projects/OWNER/REPO/issues/%d", id),
		httpmock.NewStringResponse(http.StatusOK, fmt.Sprintf(`{
			"id": %d,
			"iid": %d,
			"title": "test issue",
			"subscribed": %t,
			"issue_type": "%s",
			"created_at": "2023-05-02T10:51:26.371Z"
		}`, id, id, subscribed, issueType)),
	)
}

func mockIssuableUnsubscribe(fakeHTTP *httpmock.Mocker, id int, issueType string, subscribed bool) {
	fakeHTTP.RegisterResponder(http.MethodPost, fmt.Sprintf("/projects/OWNER/REPO/issues/%d/unsubscribe", id),
		func(req *http.Request) (*http.Response, error) {
			resp, _ := httpmock.NewStringResponse(http.StatusOK, fmt.Sprintf(`{
				"id": %d,
				"iid": %d,
				"subscribed": %t,
				"issue_type": "%s",
				"created_at": "2023-05-02T10:51:26.371Z"
			}`, id, id, subscribed, issueType))(req)

			return resp, nil
		},
	)
}

func TestIssuableUnsubscribe(t *testing.T) {
	t.Run("issue_unsubscribe", func(t *testing.T) {
		iid := 1
		fakeHTTP := httpmock.New()

		mockIssuableGet(fakeHTTP, iid, string(issuable.TypeIssue), true)
		mockIssuableUnsubscribe(fakeHTTP, iid, string(issuable.TypeIssue), false)

		output, err := runCommand(fakeHTTP, fmt.Sprint(iid), issuable.TypeIssue)

		wantOutput := heredoc.Doc(`
				- Unsubscribing from issue #1 in OWNER/REPO
				✓ Unsubscribed
				`)
		require.NoErrorf(t, err, "error running command `issue unsubscribe %d`", iid)
		require.Contains(t, output.String(), wantOutput)
		require.Empty(t, output.Stderr())
	})

	t.Run("incident_unsubscribe", func(t *testing.T) {
		iid := 2
		fakeHTTP := httpmock.New()

		mockIssuableGet(fakeHTTP, iid, string(issuable.TypeIncident), true)
		mockIssuableUnsubscribe(fakeHTTP, iid, string(issuable.TypeIncident), false)

		output, err := runCommand(fakeHTTP, fmt.Sprint(iid), issuable.TypeIncident)

		wantOutput := heredoc.Doc(`
				- Unsubscribing from incident #2 in OWNER/REPO
				✓ Unsubscribed
				`)
		require.NoErrorf(t, err, "error running command `incident unsubscribe %d`", iid)
		require.Contains(t, output.String(), wantOutput)
		require.Empty(t, output.Stderr())
	})

	t.Run("incident_unsubscribe_using_issue_command", func(t *testing.T) {
		iid := 2
		fakeHTTP := httpmock.New()

		mockIssuableGet(fakeHTTP, iid, string(issuable.TypeIncident), true)
		mockIssuableUnsubscribe(fakeHTTP, iid, string(issuable.TypeIncident), false)

		output, err := runCommand(fakeHTTP, fmt.Sprint(iid), issuable.TypeIssue)

		wantOutput := heredoc.Doc(`
				- Unsubscribing from issue #2 in OWNER/REPO
				✓ Unsubscribed
				`)
		require.NoErrorf(t, err, "error running command `issue unsubscribe %d`", iid)
		require.Contains(t, output.String(), wantOutput)
		require.Empty(t, output.Stderr())
	})

	t.Run("issue_unsubscribe_using_incident_command", func(t *testing.T) {
		iid := 1
		fakeHTTP := httpmock.New()

		mockIssuableGet(fakeHTTP, iid, string(issuable.TypeIssue), true)
		mockIssuableUnsubscribe(fakeHTTP, iid, string(issuable.TypeIssue), false)

		output, err := runCommand(fakeHTTP, fmt.Sprint(iid), issuable.TypeIncident)

		wantOutput := "Incident not found, but an issue with the provided ID exists. Run `glab issue unsubscribe <id>` to unsubscribe.\n"
		require.NoErrorf(t, err, "error running command `incident unsubscribe %d`", iid)
		require.Contains(t, output.String(), wantOutput)
		require.Empty(t, output.Stderr())
	})

	t.Run("issue_unsubscribe_from_non_subscribed_issue", func(t *testing.T) {
		iid := 3
		fakeHTTP := httpmock.New()

		mockIssuableGet(fakeHTTP, iid, string(issuable.TypeIssue), false)
		fakeHTTP.RegisterResponder(http.MethodPost, "/projects/OWNER/REPO/issues/3/unsubscribe",
			httpmock.NewStringResponse(http.StatusNotModified, ``),
		)

		output, err := runCommand(fakeHTTP, fmt.Sprint(iid), issuable.TypeIssue)

		wantOutput := heredoc.Doc(`
				- Unsubscribing from issue #3 in OWNER/REPO
				x You are not subscribed to this issue.
				`)
		require.NoErrorf(t, err, "error running command `issue unsubscribe %d`", iid)
		require.Contains(t, output.String(), wantOutput)
		require.Empty(t, output.Stderr())
	})

	t.Run("incident_unsubscribe_from_non_subscribed_incident", func(t *testing.T) {
		iid := 3
		fakeHTTP := httpmock.New()

		mockIssuableGet(fakeHTTP, iid, string(issuable.TypeIncident), false)
		fakeHTTP.RegisterResponder(http.MethodPost, "/projects/OWNER/REPO/issues/3/unsubscribe",
			httpmock.NewStringResponse(http.StatusNotModified, ``),
		)

		output, err := runCommand(fakeHTTP, fmt.Sprint(iid), issuable.TypeIncident)

		wantOutput := heredoc.Doc(`
				- Unsubscribing from incident #3 in OWNER/REPO
				x You are not subscribed to this incident.
				`)
		require.NoErrorf(t, err, "error running command `incident unsubscribe %d`", iid)
		require.Contains(t, output.String(), wantOutput)
		require.Empty(t, output.Stderr())
	})

	t.Run("issue_not_found", func(t *testing.T) {
		fakeHTTP := httpmock.New()
		fakeHTTP.RegisterResponder(http.MethodGet, "/projects/OWNER/REPO/issues/404",
			httpmock.NewStringResponse(http.StatusNotFound, `{"message": "404 not found"}`),
		)

		iid := 404
		_, err := runCommand(fakeHTTP, fmt.Sprint(iid), issuable.TypeIssue)

		require.Contains(t, err.Error(), "404 Not Found")
	})
}
