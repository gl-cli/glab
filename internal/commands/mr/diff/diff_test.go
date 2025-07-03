package diff

import (
	"bytes"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/MakeNowJust/heredoc/v2"
	"github.com/google/shlex"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gitlab.com/gitlab-org/cli/internal/git"
	"gitlab.com/gitlab-org/cli/internal/glinstance"
	"gitlab.com/gitlab-org/cli/internal/glrepo"
	"gitlab.com/gitlab-org/cli/internal/testing/cmdtest"
	"gitlab.com/gitlab-org/cli/internal/testing/httpmock"
	"gitlab.com/gitlab-org/cli/test"
)

func Test_NewCmdDiff(t *testing.T) {
	tests := []struct {
		name    string
		args    string
		isTTY   bool
		want    options
		wantErr string
	}{
		{
			name:  "number argument",
			args:  "123",
			isTTY: true,
			want: options{
				args:     []string{"123"},
				useColor: "auto",
			},
		},
		{
			name:  "no argument",
			args:  "",
			isTTY: true,
			want: options{
				useColor: "auto",
			},
		},
		{
			name:  "no color when redirected",
			args:  "",
			isTTY: false,
			want: options{
				useColor: "never",
			},
		},
		{
			name:    "no argument with --repo override",
			args:    "-R owner/repo",
			isTTY:   true,
			wantErr: "argument required when using the --repo flag.",
		},
		{
			name:    "invalid --color argument",
			args:    "--color doublerainbow",
			isTTY:   true,
			wantErr: `did not understand color: "doublerainbow". Expected one of 'always', 'never', or 'auto'.`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ios, _, _, _ := cmdtest.TestIOStreams(cmdtest.WithTestIOStreamsAsTTY(tt.isTTY))

			f := cmdtest.NewTestFactory(ios)

			var opts *options
			cmd := NewCmdDiff(f, func(o *options) error {
				opts = o
				return nil
			})
			cmd.PersistentFlags().StringP("repo", "R", "", "")

			argv, err := shlex.Split(tt.args)
			require.NoError(t, err)
			cmd.SetArgs(argv)

			cmd.SetIn(&bytes.Buffer{})
			cmd.SetOut(io.Discard)
			cmd.SetErr(io.Discard)

			_, err = cmd.ExecuteC()
			if tt.wantErr != "" {
				require.EqualError(t, err, tt.wantErr)
				return
			} else {
				require.NoError(t, err)
			}

			assert.Equal(t, tt.want.args, opts.args)
			assert.Equal(t, tt.want.useColor, opts.useColor)
		})
	}
}

func runCommand(t *testing.T, rt http.RoundTripper, isTTY bool, cli string) (*test.CmdOut, error) {
	ios, _, stdout, stderr := cmdtest.TestIOStreams(cmdtest.WithTestIOStreamsAsTTY(isTTY))

	factory := cmdtest.NewTestFactory(ios,
		cmdtest.WithGitLabClient(cmdtest.NewTestApiClient(t, &http.Client{Transport: rt}, "", glinstance.DefaultHostname).Lab()),
	)

	factory.RemotesStub = func() (glrepo.Remotes, error) {
		return glrepo.Remotes{
			{
				Remote: &git.Remote{Name: "origin"},
				Repo:   glrepo.New("OWNER", "REPO", glinstance.DefaultHostname),
			},
		}, nil
	}
	factory.BranchStub = func() (string, error) {
		return "feature", nil
	}

	cmd := NewCmdDiff(factory, nil)

	return cmdtest.ExecuteCommand(cmd, cli, stdout, stderr)
}

func TestMRDiff_raw(t *testing.T) {
	fakeHTTP := &httpmock.Mocker{
		MatchURL: httpmock.PathAndQuerystring,
	}
	defer fakeHTTP.Verify(t)

	fakeHTTP.RegisterResponder(
		http.MethodGet,
		`https://gitlab.com/api/v4/projects/OWNER%2FREPO/merge_requests/123`,
		MRGetResponse(),
	)

	rawDiff := heredoc.Doc(`
	diff --git a/file.txt b/file.txt
	index 123..456 100644
	--- a/file.txt
	+++ b/file.txt
	@@ -1 +1 @@
	-old line
	+new line`)

	fakeHTTP.RegisterResponder(http.MethodGet, `https://gitlab.com/api/v4/projects/OWNER%2FREPO/merge_requests/123/raw_diffs`,
		httpmock.NewStringResponse(http.StatusOK, rawDiff))

	output, err := runCommand(t, fakeHTTP, false, "123 --raw")
	require.NoError(t, err)
	assert.Equal(t, rawDiff, output.String())
	assert.Empty(t, output.Stderr())
}

func TestPRDiff_no_current_mr(t *testing.T) {
	fakeHTTP := &httpmock.Mocker{
		MatchURL: httpmock.PathAndQuerystring,
	}
	defer fakeHTTP.Verify(t)

	fakeHTTP.RegisterResponder(http.MethodGet, `https://gitlab.com/api/v4/projects/OWNER/REPO/merge_requests?per_page=30&source_branch=feature`,
		httpmock.NewStringResponse(http.StatusOK, `[]`))

	_, err := runCommand(t, fakeHTTP, false, "")
	if err == nil {
		t.Fatal("expected error")
	}
	assert.Equal(t, `no open merge request available for "feature"`, err.Error())
}

func TestMRDiff_argument_not_found(t *testing.T) {
	fakeHTTP := &httpmock.Mocker{
		MatchURL: httpmock.PathOnly,
	}
	defer fakeHTTP.Verify(t)

	fakeHTTP.RegisterResponder(http.MethodGet, `/projects/OWNER/REPO/merge_requests/123`, MRGetResponse())

	fakeHTTP.RegisterResponder(http.MethodGet, `/projects/OWNER/REPO/merge_requests/123/versions`,
		httpmock.NewStringResponse(http.StatusNotFound, `{"message":"404 Not Found"}`))

	output, err := runCommand(t, fakeHTTP, false, "123")
	if err == nil {
		t.Fatal("expected error", err)
	}

	assert.Empty(t, output.String())
	assert.Empty(t, output.Stderr())
	assert.Equal(t, `could not find merge request diffs: 404 Not Found`, err.Error())
}

func TestMRDiff_notty(t *testing.T) {
	fakeHTTP := &httpmock.Mocker{
		MatchURL: httpmock.PathAndQuerystring,
	}

	defer fakeHTTP.Verify(t)

	fakeHTTP.RegisterResponder(http.MethodGet, `https://gitlab.com/api/v4/projects/OWNER%2FREPO/merge_requests?per_page=30&source_branch=feature`,
		httpmock.NewStringResponse(http.StatusOK, `[{
    "id": 123,
    "iid": 123,
    "project_id": 3,
    "title": "test1",
    "description": "fixed login page css paddings",
    "state": "merged"}]`))

	fakeHTTP.RegisterResponder(http.MethodGet, `https://gitlab.com/api/v4/projects/OWNER%2FREPO/merge_requests/123`,
		MRGetResponse())

	testDiff := DiffTest(fakeHTTP)
	output, err := runCommand(t, fakeHTTP, false, "")
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}
	if diff := strings.Contains(testDiff, output.String()); diff {
		t.Errorf("command output did not match:\n%v", diff)
	}
}

func TestMRDiff_tty(t *testing.T) {
	fakeHTTP := &httpmock.Mocker{
		MatchURL: httpmock.PathAndQuerystring,
	}

	defer fakeHTTP.Verify(t)

	fakeHTTP.RegisterResponder(http.MethodGet, `https://gitlab.com/api/v4/projects/OWNER%2FREPO/merge_requests?per_page=30&source_branch=feature`,
		httpmock.NewStringResponse(http.StatusOK, `[{
    "id": 123,
    "iid": 123,
    "project_id": 3,
    "title": "test1",
    "description": "fixed login page css paddings",
    "state": "merged"}]`))

	fakeHTTP.RegisterResponder(http.MethodGet, `https://gitlab.com/api/v4/projects/OWNER%2FREPO/merge_requests/123`,
		MRGetResponse())

	DiffTest(fakeHTTP)
	output, err := runCommand(t, fakeHTTP, true, "")
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}
	assert.Contains(t, output.String(), "\x1b[m\n\x1b[32m+FITNESS")
}

func DiffTest(fakeHTTP *httpmock.Mocker) string {
	fakeHTTP.RegisterResponder(http.MethodGet, `https://gitlab.com/api/v4/projects/OWNER%2FREPO/merge_requests/123/versions`,
		httpmock.NewStringResponse(http.StatusOK, `[{
  "id": 110,
  "head_commit_sha": "33e2ee8579fda5bc36accc9c6fbd0b4fefda9e30",
  "base_commit_sha": "eeb57dffe83deb686a60a71c16c32f71046868fd",
  "start_commit_sha": "eeb57dffe83deb686a60a71c16c32f71046868fd",
  "created_at": "2016-07-26T14:44:48.926Z",
  "merge_request_id": 105,
  "state": "collected",
  "real_size": "1"
}, {
  "id": 108,
  "head_commit_sha": "3eed087b29835c48015768f839d76e5ea8f07a24",
  "base_commit_sha": "eeb57dffe83deb686a60a71c16c32f71046868fd",
  "start_commit_sha": "eeb57dffe83deb686a60a71c16c32f71046868fd",
  "created_at": "2016-07-25T14:21:33.028Z",
  "merge_request_id": 105,
  "state": "collected",
  "real_size": "1",
  "patch_id_sha": "72c30d1f0115fc1d2bb0b29b24dc2982cbcdfd32"
}]`))

	fakeHTTP.RegisterResponder(http.MethodGet, `https://gitlab.com/api/v4/projects/OWNER%2FREPO/merge_requests/123/versions/110`,
		httpmock.NewStringResponse(http.StatusOK, `{
  "id": 110,
  "head_commit_sha": "33e2ee8579fda5bc36accc9c6fbd0b4fefda9e30",
  "base_commit_sha": "eeb57dffe83deb686a60a71c16c32f71046868fd",
  "start_commit_sha": "eeb57dffe83deb686a60a71c16c32f71046868fd",
  "created_at": "2016-07-26T14:44:48.926Z",
  "merge_request_id": 105,
  "state": "collected",
  "real_size": "1",
  "commits": [{
    "id": "33e2ee8579fda5bc36accc9c6fbd0b4fefda9e30",
    "short_id": "33e2ee85",
    "title": "Change year to 2018",
    "author_name": "Administrator",
    "author_email": "admin@example.com",
    "created_at": "2016-07-26T17:44:29.000+03:00",
    "message": "Change year to 2018"
  }, {
    "id": "aa24655de48b36335556ac8a3cd8bb521f977cbd",
    "short_id": "aa24655d",
    "title": "Update LICENSE",
    "author_name": "Administrator",
    "author_email": "admin@example.com",
    "created_at": "2016-07-25T17:21:53.000+03:00",
    "message": "Update LICENSE"
  }, {
    "id": "3eed087b29835c48015768f839d76e5ea8f07a24",
    "short_id": "3eed087b",
    "title": "Add license",
    "author_name": "Administrator",
    "author_email": "admin@example.com",
    "created_at": "2016-07-25T17:21:20.000+03:00",
    "message": "Add license"
  }],
  "diffs": [{
    "old_path": "LICENSE.md",
    "new_path": "LICENSE",
    "a_mode": "0",
    "b_mode": "100644",
    "diff": "--- /dev/null\n+++ b/LICENSE\n@@ -0,0 +1,21 @@\n+The MIT License (MIT)\n+\n+Copyright (c) 2018 Administrator\n+\n+Permission is hereby granted, free of charge, to any person obtaining a copy\n+of this software and associated documentation files (the \"Software\"), to deal\n+in the Software without restriction, including without limitation the rights\n+to use, copy, modify, merge, publish, distribute, sublicense, and/or sell\n+copies of the Software, and to permit persons to whom the Software is\n+furnished to do so, subject to the following conditions:\n+\n+The above copyright notice and this permission notice shall be included in all\n+copies or substantial portions of the Software.\n+\n+THE SOFTWARE IS PROVIDED \"AS IS\", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR\n+IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,\n+FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE\n+AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER\n+LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,\n+OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE\n+SOFTWARE.\n",
    "new_file": true,
    "renamed_file": true,
    "deleted_file": false
  }]
}`))
	return "--- /dev/null\n+++ b/LICENSE\n@@ -0,0 +1,21 @@\n+The MIT License (MIT)\n+\n+Copyright (c) 2018 Administrator\n+\n+Permission is hereby granted, free of charge, to any person obtaining a copy\n+of this software and associated documentation files (the \"Software\"), to deal\n+in the Software without restriction, including without limitation the rights\n+to use, copy, modify, merge, publish, distribute, sublicense, and/or sell\n+copies of the Software, and to permit persons to whom the Software is\n+furnished to do so, subject to the following conditions:\n+\n+The above copyright notice and this permission notice shall be included in all\n+copies or substantial portions of the Software.\n+\n+THE SOFTWARE IS PROVIDED \"AS IS\", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR\n+IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,\n+FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE\n+AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER\n+LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,\n+OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE\n+SOFTWARE.\n"
}

func TestMRDiff_no_diffs_found(t *testing.T) {
	fakeHTTP := &httpmock.Mocker{
		MatchURL: httpmock.PathAndQuerystring,
	}
	defer fakeHTTP.Verify(t)

	fakeHTTP.RegisterResponder(http.MethodGet, `https://gitlab.com/api/v4/projects/OWNER%2FREPO/merge_requests?per_page=30&source_branch=feature`,
		httpmock.NewStringResponse(http.StatusOK, `[{
	"id": 123,
	"iid": 123,
	"project_id": 3,
	"title": "test1",
	"description": "fixed login page css paddings",
	"state": "merged"}]`))

	fakeHTTP.RegisterResponder(http.MethodGet, `https://gitlab.com/api/v4/projects/OWNER%2FREPO/merge_requests/123`,
		MRGetResponse())

	EmptyDiffsTest(fakeHTTP)

	_, err := runCommand(t, fakeHTTP, false, "")
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	assert.Error(t, err, "no merge request diffs found")
}

func EmptyDiffsTest(fakeHTTP *httpmock.Mocker) {
	fakeHTTP.RegisterResponder(http.MethodGet, `https://gitlab.com/api/v4/projects/OWNER%2FREPO/merge_requests/123/versions`,
		httpmock.NewStringResponse(http.StatusOK, `[]`))
}

func MRGetResponse() httpmock.Responder {
	return httpmock.NewStringResponse(http.StatusOK, `{
		"id": 123,
		"iid": 123,
		"project_id": 3,
		"title": "test1",
		"description": "fixed login page css paddings",
		"state": "merged"
	}`)
}
