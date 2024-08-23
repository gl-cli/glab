package note

import (
	"fmt"
	"net/http"
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
	"gitlab.com/gitlab-org/cli/commands/cmdtest"
	"gitlab.com/gitlab-org/cli/commands/issuable"
	"gitlab.com/gitlab-org/cli/internal/config"
	"gitlab.com/gitlab-org/cli/pkg/git"
	"gitlab.com/gitlab-org/cli/pkg/httpmock"
	"gitlab.com/gitlab-org/cli/pkg/prompt"
	"gitlab.com/gitlab-org/cli/test"
)

func runCommand(rt http.RoundTripper, isTTY bool, cli string, issueType issuable.IssueType) (*test.CmdOut, error) {
	ios, _, stdout, stderr := cmdtest.InitIOStreams(isTTY, "")
	factory := cmdtest.InitFactory(ios, rt)
	factory.Branch = git.CurrentBranch
	factory.Config = func() (config.Config, error) {
		return config.NewFromString("editor: vi"), nil
	}

	// TODO: shouldn't be there but the stub doesn't work without it
	_, _ = factory.HttpClient()

	cmd := NewCmdNote(factory, issueType)

	return cmdtest.ExecuteCommand(cmd, cli, stdout, stderr)
}

func Test_NewCmdNote(t *testing.T) {
	fakeHTTP := httpmock.New()
	defer fakeHTTP.Verify(t)

	commands := []struct {
		name      string
		issueType issuable.IssueType
	}{
		{"issue", issuable.TypeIssue},
		{"incident", issuable.TypeIncident},
	}

	for _, cc := range commands {
		t.Run("--message flag specified", func(t *testing.T) {
			fakeHTTP.RegisterResponder(http.MethodPost, "/projects/OWNER/REPO/issues/1/notes",
				httpmock.NewStringResponse(http.StatusCreated, `
				{
					"id": 301,
					"created_at": "2013-10-02T08:57:14Z",
					"updated_at": "2013-10-02T08:57:14Z",
					"system": false,
					"noteable_id": 1,
					"noteable_type": "MergeRequest",
					"noteable_iid": 1
				}
			`))

			fakeHTTP.RegisterResponder(http.MethodGet, "/projects/OWNER/REPO/issues/1",
				httpmock.NewStringResponse(http.StatusOK, fmt.Sprintf(`
				{
					"id": 1,
					"iid": 1,
					"issue_type": "%s",
					"web_url": "https://gitlab.com/OWNER/REPO/issues/1"
				}
			`, cc.issueType)))

			// glab issue note 1 --message "Here is my note"
			// glab incident note 1 --message "Here is my note"
			output, err := runCommand(fakeHTTP, true, `1 --message "Here is my note"`, cc.issueType)
			if err != nil {
				t.Error(err)
				return
			}
			assert.Equal(t, output.Stderr(), "")
			assert.Equal(t, output.String(), "https://gitlab.com/OWNER/REPO/issues/1#note_301\n")
		})

		t.Run("issue not found", func(t *testing.T) {
			fakeHTTP.RegisterResponder(http.MethodGet, "/projects/OWNER/REPO/issues/122",
				httpmock.NewStringResponse(http.StatusNotFound, `
				{
					"message": "issue not found"
				}
			`))

			// glab issue note 1 --message "Here is my note"
			// glab incident note 1 --message "Here is my note"
			_, err := runCommand(fakeHTTP, true, `122`, cc.issueType)
			assert.NotNil(t, err)
			assert.Equal(t, "404 Not Found", err.Error())
		})
	}
}

func Test_NewCmdNote_error(t *testing.T) {
	fakeHTTP := httpmock.New()
	defer fakeHTTP.Verify(t)

	commands := []struct {
		name      string
		issueType issuable.IssueType
	}{
		{"issue", issuable.TypeIssue},
		{"incident", issuable.TypeIncident},
	}

	for _, cc := range commands {
		t.Run("note could not be created", func(t *testing.T) {
			fakeHTTP.RegisterResponder(http.MethodPost, "/projects/OWNER/REPO/issues/1/notes",
				httpmock.NewStringResponse(http.StatusUnauthorized, `
				{
					"message": "Unauthorized"
				}
			`))

			fakeHTTP.RegisterResponder(http.MethodGet, "/projects/OWNER/REPO/issues/1",
				httpmock.NewStringResponse(http.StatusOK, fmt.Sprintf(`
				{
					"id": 1,
					"iid": 1,
					"issue_type": "%s",
					"web_url": "https://gitlab.com/OWNER/REPO/issues/1"
				}
			`, cc.issueType)))

			// glab issue note 1 --message "Here is my note"
			// glab incident note 1 --message "Here is my note"
			_, err := runCommand(fakeHTTP, true, `1 -m "Some message"`, cc.issueType)
			assert.NotNil(t, err)
			assert.Equal(t, "POST https://gitlab.com/api/v4/projects/OWNER/REPO/issues/1/notes: 401 {message: Unauthorized}", err.Error())
		})
	}

	t.Run("using incident note command with issue ID", func(t *testing.T) {
		fakeHTTP.RegisterResponder(http.MethodGet, "/projects/OWNER/REPO/issues/1",
			httpmock.NewStringResponse(http.StatusOK, `
				{
					"id": 1,
					"iid": 1,
					"issue_type": "issue",
					"web_url": "https://gitlab.com/OWNER/REPO/issues/1"
				}
			`))

		output, err := runCommand(fakeHTTP, true, `1 -m "Some message"`, issuable.TypeIncident)
		assert.Nil(t, err)
		assert.Equal(t, "Incident not found, but an issue with the provided ID exists. Run `glab issue comment <id>` to comment.\n", output.String())
	})
}

func Test_IssuableNoteCreate_prompt(t *testing.T) {
	fakeHTTP := httpmock.New()
	defer fakeHTTP.Verify(t)

	commands := []struct {
		name      string
		issueType issuable.IssueType
	}{
		{"issue", issuable.TypeIssue},
		{"incident", issuable.TypeIncident},
	}

	for _, cc := range commands {
		t.Run("message provided", func(t *testing.T) {
			fakeHTTP.RegisterResponder(http.MethodPost, "/projects/OWNER/REPO/issues/1/notes",
				httpmock.NewStringResponse(http.StatusCreated, `
				{
					"id": 301,
					"created_at": "2013-10-02T08:57:14Z",
					"updated_at": "2013-10-02T08:57:14Z",
					"system": false,
					"noteable_id": 1,
					"noteable_type": "MergeRequest",
					"noteable_iid": 1
				}
			`))

			fakeHTTP.RegisterResponder(http.MethodGet, "/projects/OWNER/REPO/issues/1",
				httpmock.NewStringResponse(http.StatusOK, fmt.Sprintf(`
				{
					"id": 1,
					"iid": 1,
					"issue_type": "%s",
					"web_url": "https://gitlab.com/OWNER/REPO/issues/1"
				}
			`, cc.issueType)))
			as, teardown := prompt.InitAskStubber()
			defer teardown()
			as.StubOne("some note message")

			// glab issue note 1
			// glab incident note 1
			output, err := runCommand(fakeHTTP, true, `1`, cc.issueType)

			// get the editor used
			notePrompt := *as.AskOnes[0]
			actualEditor := reflect.ValueOf(notePrompt).Elem().FieldByName("EditorCommand").String()

			if err != nil {
				t.Error(err)
				return
			}
			assert.Equal(t, "", output.Stderr())
			assert.Equal(t, "https://gitlab.com/OWNER/REPO/issues/1#note_301\n", output.String())
			assert.Equal(t, "vi", actualEditor)
		})

		tests := []struct {
			name    string
			message string
		}{
			{"message is empty", ""},
			{"message contains only spaces", "   "},
			{"message contains only line breaks", "\n\n"},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				fakeHTTP.RegisterResponder(http.MethodGet, "/projects/OWNER/REPO/issues/1",
					httpmock.NewStringResponse(http.StatusOK, fmt.Sprintf(`
					{
						"id": 1,
						"iid": 1,
						"issue_type": "%s",
						"web_url": "https://gitlab.com/OWNER/REPO/issues/1"
					}
				`, cc.issueType)))

				as, teardown := prompt.InitAskStubber()
				defer teardown()
				as.StubOne(tt.message)

				_, err := runCommand(fakeHTTP, true, `1`, cc.issueType)
				if err == nil {
					t.Error("expected error")
					return
				}
				assert.Equal(t, "aborted... Note is empty.", err.Error())
			})
		}
	}
}
