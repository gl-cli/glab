package note

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"gitlab.com/gitlab-org/cli/internal/git"
	"gitlab.com/gitlab-org/cli/internal/glinstance"
	"gitlab.com/gitlab-org/cli/internal/prompt"
	"gitlab.com/gitlab-org/cli/internal/testing/cmdtest"
	"gitlab.com/gitlab-org/cli/internal/testing/httpmock"
	"gitlab.com/gitlab-org/cli/test"
)

func TestMain(m *testing.M) {
	cmdtest.InitTest(m, "mr_note_create_test")
}

func runCommand(t *testing.T, rt http.RoundTripper, cli string) (*test.CmdOut, error) {
	ios, _, stdout, stderr := cmdtest.TestIOStreams(cmdtest.WithTestIOStreamsAsTTY(true))

	factory := cmdtest.NewTestFactory(ios,
		cmdtest.WithGitLabClient(cmdtest.NewTestApiClient(t, &http.Client{Transport: rt}, "", glinstance.DefaultHostname).Lab()),
	)
	factory.BranchStub = git.CurrentBranch

	cmd := NewCmdNote(factory)

	return cmdtest.ExecuteCommand(cmd, cli, stdout, stderr)
}

func Test_NewCmdNote(t *testing.T) {
	fakeHTTP := httpmock.New()
	defer fakeHTTP.Verify(t)

	t.Run("--message flag specified", func(t *testing.T) {
		fakeHTTP.RegisterResponder(http.MethodPost, "/projects/OWNER/REPO/merge_requests/1/notes",
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

		fakeHTTP.RegisterResponder(http.MethodGet, "/projects/OWNER/REPO/merge_requests/1",
			httpmock.NewStringResponse(http.StatusOK, `
		{
  			"id": 1,
  			"iid": 1,
			"web_url": "https://gitlab.com/OWNER/REPO/merge_requests/1"
		}
	`))

		// glab mr note 1 --message "Here is my note"
		output, err := runCommand(t, fakeHTTP, `1 --message "Here is my note"`)
		if err != nil {
			t.Error(err)
			return
		}
		assert.Equal(t, output.Stderr(), "")
		assert.Equal(t, output.String(), "https://gitlab.com/OWNER/REPO/merge_requests/1#note_301\n")
	})

	t.Run("merge request not found", func(t *testing.T) {
		fakeHTTP.RegisterResponder(http.MethodGet, "/projects/OWNER/REPO/merge_requests/122",
			httpmock.NewStringResponse(http.StatusNotFound, `
		{
  			"message": "merge request not found"
		}
	`))

		// glab mr note 1 --message "Here is my note"
		_, err := runCommand(t, fakeHTTP, `122`)
		assert.NotNil(t, err)
		assert.Equal(t, "failed to get merge request 122: 404 Not Found", err.Error())
	})
}

func Test_NewCmdNote_error(t *testing.T) {
	fakeHTTP := httpmock.New()
	defer fakeHTTP.Verify(t)

	t.Run("note could not be created", func(t *testing.T) {
		fakeHTTP.RegisterResponder(http.MethodPost, "/projects/OWNER/REPO/merge_requests/1/notes",
			httpmock.NewStringResponse(http.StatusUnauthorized, `
		{
			"message": "Unauthorized"
		}
	`))

		fakeHTTP.RegisterResponder(http.MethodGet, "/projects/OWNER/REPO/merge_requests/1",
			httpmock.NewStringResponse(http.StatusOK, `
		{
  			"id": 1,
  			"iid": 1,
			"web_url": "https://gitlab.com/OWNER/REPO/merge_requests/1"
		}
	`))

		// glab mr note 1 --message "Here is my note"
		_, err := runCommand(t, fakeHTTP, `1 -m "Some message"`)
		assert.NotNil(t, err)
		assert.Equal(t, "POST https://gitlab.com/api/v4/projects/OWNER%2FREPO/merge_requests/1/notes: 401 {message: Unauthorized}", err.Error())
	})
}

func Test_mrNoteCreate_prompt(t *testing.T) {
	fakeHTTP := httpmock.New()
	defer fakeHTTP.Verify(t)

	t.Run("message provided", func(t *testing.T) {
		fakeHTTP.RegisterResponder(http.MethodPost, "/projects/OWNER/REPO/merge_requests/1/notes",
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

		fakeHTTP.RegisterResponder(http.MethodGet, "/projects/OWNER/REPO/merge_requests/1",
			httpmock.NewStringResponse(http.StatusOK, `
		{
  			"id": 1,
  			"iid": 1,
			"web_url": "https://gitlab.com/OWNER/REPO/merge_requests/1"
		}
	`))
		as, teardown := prompt.InitAskStubber()
		defer teardown()
		as.StubOne("some note message")

		// glab mr note 1
		output, err := runCommand(t, fakeHTTP, `1`)
		if err != nil {
			t.Error(err)
			return
		}
		assert.Equal(t, output.Stderr(), "")
		assert.Equal(t, output.String(), "https://gitlab.com/OWNER/REPO/merge_requests/1#note_301\n")
	})

	t.Run("message is empty", func(t *testing.T) {
		fakeHTTP.RegisterResponder(http.MethodGet, "/projects/OWNER/REPO/merge_requests/1",
			httpmock.NewStringResponse(http.StatusOK, `
		{
  			"id": 1,
  			"iid": 1,
			"web_url": "https://gitlab.com/OWNER/REPO/merge_requests/1"
		}
	`))

		as, teardown := prompt.InitAskStubber()
		defer teardown()
		as.StubOne("")

		// glab mr note 1
		_, err := runCommand(t, fakeHTTP, `1`)
		if err == nil {
			t.Error("expected error")
			return
		}
		assert.Equal(t, err.Error(), "aborted... Note has an empty message.")
	})
}

func Test_mrNoteCreate_no_duplicate(t *testing.T) {
	fakeHTTP := httpmock.New()
	defer fakeHTTP.Verify(t)

	t.Run("message provided", func(t *testing.T) {
		fakeHTTP.RegisterResponder(http.MethodGet, "/projects/OWNER/REPO/merge_requests/1",
			httpmock.NewStringResponse(http.StatusOK, `
		{
  			"id": 1,
  			"iid": 1,
			"web_url": "https://gitlab.com/OWNER/REPO/merge_requests/1"
		}
	`))

		fakeHTTP.RegisterResponder(http.MethodGet, "/projects/OWNER/REPO/merge_requests/1/notes",
			httpmock.NewStringResponse(http.StatusOK, `
		[
			{"id": 0, "body": "aaa"},
			{"id": 111, "body": "bbb"},
			{"id": 222, "body": "some note message"},
			{"id": 333, "body": "ccc"}
		]
	`))
		as, teardown := prompt.InitAskStubber()
		defer teardown()
		as.StubOne("some note message")

		// glab mr note 1
		output, err := runCommand(t, fakeHTTP, `1 --unique`)
		if err != nil {
			t.Error(err)
			return
		}
		println(output.String())
		assert.Equal(t, output.Stderr(), "")
		assert.Equal(t, output.String(), "https://gitlab.com/OWNER/REPO/merge_requests/1#note_222\n")
	})
}
