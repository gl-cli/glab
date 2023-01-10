package note

import (
	"net/http"
	"testing"

	"github.com/alecthomas/assert"
	"gitlab.com/gitlab-org/cli/commands/cmdtest"
	"gitlab.com/gitlab-org/cli/pkg/git"
	"gitlab.com/gitlab-org/cli/pkg/httpmock"
	"gitlab.com/gitlab-org/cli/pkg/prompt"
	"gitlab.com/gitlab-org/cli/test"
)

func runCommand(rt http.RoundTripper, isTTY bool, cli string) (*test.CmdOut, error) {
	ios, _, stdout, stderr := cmdtest.InitIOStreams(isTTY, "")
	factory := cmdtest.InitFactory(ios, rt)
	factory.Branch = git.CurrentBranch

	// TODO: shouldn't be there but the stub doesn't work without it
	_, _ = factory.HttpClient()

	cmd := NewCmdNote(factory)

	return cmdtest.ExecuteCommand(cmd, cli, stdout, stderr)
}

func Test_NewCmdNote(t *testing.T) {
	fakeHTTP := httpmock.New()
	defer fakeHTTP.Verify(t)

	t.Run("--message flag specified", func(t *testing.T) {
		fakeHTTP.RegisterResponder("POST", "/projects/OWNER/REPO/issues/1/notes",
			httpmock.NewStringResponse(201, `
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

		fakeHTTP.RegisterResponder("GET", "/projects/OWNER/REPO/issues/1",
			httpmock.NewStringResponse(200, `
		{
  			"id": 1,
  			"iid": 1,
			"web_url": "https://gitlab.com/OWNER/REPO/issues/1"
		}
	`))

		// glab mr note 1 --message "Here is my note"
		output, err := runCommand(fakeHTTP, true, `1 --message "Here is my note"`)
		if err != nil {
			t.Error(err)
			return
		}
		assert.Equal(t, output.Stderr(), "")
		assert.Equal(t, output.String(), "https://gitlab.com/OWNER/REPO/issues/1#note_301\n")
	})

	t.Run("issue not found", func(t *testing.T) {
		fakeHTTP.RegisterResponder("GET", "/projects/OWNER/REPO/issues/122",
			httpmock.NewStringResponse(404, `
		{
  			"message" : "issue not found"
		}
	`))

		// glab mr note 1 --message "Here is my note"
		_, err := runCommand(fakeHTTP, true, `122`)
		assert.NotNil(t, err)
		assert.Equal(t, "GET https://gitlab.com/api/v4/projects/OWNER/REPO/issues/122: 404 {message: issue not found}", err.Error())
	})
}

func Test_NewCmdNote_error(t *testing.T) {
	fakeHTTP := httpmock.New()
	defer fakeHTTP.Verify(t)

	t.Run("note could not be created", func(t *testing.T) {
		fakeHTTP.RegisterResponder("POST", "/projects/OWNER/REPO/issues/1/notes",
			httpmock.NewStringResponse(401, `
		{
			"message" : "Unauthorized"
		}
	`))

		fakeHTTP.RegisterResponder("GET", "/projects/OWNER/REPO/issues/1",
			httpmock.NewStringResponse(200, `
		{
  			"id": 1,
  			"iid": 1,
			"web_url": "https://gitlab.com/OWNER/REPO/issues/1"
		}
	`))

		// glab mr note 1 --message "Here is my note"
		_, err := runCommand(fakeHTTP, true, `1 -m "Some message"`)
		assert.NotNil(t, err)
		assert.Equal(t, "POST https://gitlab.com/api/v4/projects/OWNER/REPO/issues/1/notes: 401 {message: Unauthorized}", err.Error())
	})
}

func Test_mrNoteCreate_prompt(t *testing.T) {
	fakeHTTP := httpmock.New()
	defer fakeHTTP.Verify(t)

	t.Run("message provided", func(t *testing.T) {
		fakeHTTP.RegisterResponder("POST", "/projects/OWNER/REPO/issues/1/notes",
			httpmock.NewStringResponse(201, `
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

		fakeHTTP.RegisterResponder("GET", "/projects/OWNER/REPO/issues/1",
			httpmock.NewStringResponse(200, `
		{
  			"id": 1,
  			"iid": 1,
			"web_url": "https://gitlab.com/OWNER/REPO/issues/1"
		}
	`))
		as, teardown := prompt.InitAskStubber()
		defer teardown()
		as.StubOne("some note message")

		// glab mr note 1
		output, err := runCommand(fakeHTTP, true, `1`)
		if err != nil {
			t.Error(err)
			return
		}
		assert.Equal(t, output.Stderr(), "")
		assert.Equal(t, output.String(), "https://gitlab.com/OWNER/REPO/issues/1#note_301\n")
	})

	t.Run("message is empty", func(t *testing.T) {
		fakeHTTP.RegisterResponder("GET", "/projects/OWNER/REPO/issues/1",
			httpmock.NewStringResponse(200, `
		{
  			"id": 1,
  			"iid": 1,
			"web_url": "https://gitlab.com/OWNER/REPO/issues/1"
		}
	`))

		as, teardown := prompt.InitAskStubber()
		defer teardown()
		as.StubOne("")

		// glab mr note 1
		_, err := runCommand(fakeHTTP, true, `1`)
		if err == nil {
			t.Error("expected error")
			return
		}
		assert.Equal(t, "aborted... Note is empty", err.Error())
	})
}
