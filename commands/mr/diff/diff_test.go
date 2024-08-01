package diff

import (
	"bytes"
	"io"
	"net/http"
	"strings"
	"testing"

	"gitlab.com/gitlab-org/cli/pkg/iostreams"

	"github.com/google/shlex"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gitlab.com/gitlab-org/cli/commands/cmdtest"
	"gitlab.com/gitlab-org/cli/commands/cmdutils"
	"gitlab.com/gitlab-org/cli/internal/glrepo"
	"gitlab.com/gitlab-org/cli/pkg/git"
	"gitlab.com/gitlab-org/cli/pkg/httpmock"
	"gitlab.com/gitlab-org/cli/test"
)

func Test_NewCmdDiff(t *testing.T) {
	tests := []struct {
		name    string
		args    string
		isTTY   bool
		want    DiffOptions
		wantErr string
	}{
		{
			name:  "number argument",
			args:  "123",
			isTTY: true,
			want: DiffOptions{
				Args:     []string{"123"},
				UseColor: "auto",
			},
		},
		{
			name:  "no argument",
			args:  "",
			isTTY: true,
			want: DiffOptions{
				UseColor: "auto",
			},
		},
		{
			name:  "no color when redirected",
			args:  "",
			isTTY: false,
			want: DiffOptions{
				UseColor: "never",
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
			ios, _, _, _ := iostreams.Test()
			ios.IsaTTY = tt.isTTY
			ios.IsInTTY = tt.isTTY
			ios.IsErrTTY = tt.isTTY

			f := &cmdutils.Factory{
				IO: ios,
			}

			var opts *DiffOptions
			cmd := NewCmdDiff(f, func(o *DiffOptions) error {
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

			assert.Equal(t, tt.want.Args, opts.Args)
			assert.Equal(t, tt.want.UseColor, opts.UseColor)
		})
	}
}

func runCommand(rt http.RoundTripper, remotes glrepo.Remotes, isTTY bool, cli string) (*test.CmdOut, error) {
	ios, _, stdout, stderr := cmdtest.InitIOStreams(isTTY, "")

	factory := cmdtest.InitFactory(ios, rt)
	_, _ = factory.HttpClient()

	factory.Remotes = func() (glrepo.Remotes, error) {
		if remotes == nil {
			return glrepo.Remotes{
				{
					Remote: &git.Remote{Name: "origin"},
					Repo:   glrepo.New("OWNER", "REPO"),
				},
			}, nil
		}
		return remotes, nil
	}
	factory.Branch = func() (string, error) {
		return "feature", nil
	}

	cmd := NewCmdDiff(factory, nil)

	return cmdtest.ExecuteCommand(cmd, cli, stdout, stderr)
}

func TestPRDiff_no_current_mr(t *testing.T) {
	fakeHTTP := &httpmock.Mocker{
		MatchURL: httpmock.PathAndQuerystring,
	}
	defer fakeHTTP.Verify(t)

	fakeHTTP.RegisterResponder(http.MethodGet, `https://gitlab.com/api/v4/projects/OWNER/REPO/merge_requests?per_page=30&source_branch=feature`,
		httpmock.NewStringResponse(http.StatusOK, `[]`))

	_, err := runCommand(fakeHTTP, nil, false, "")
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

	fakeHTTP.RegisterResponder(http.MethodGet, `/projects/OWNER/REPO/merge_requests/123`,
		httpmock.NewStringResponse(http.StatusOK, `{
			"id": 123,
			"iid": 123,
			"project_id": 3,
			"title": "test1",
			"description": "fixed login page css paddings",
			"state": "merged"
		}`))

	fakeHTTP.RegisterResponder(http.MethodGet, `/projects/OWNER/REPO/merge_requests/123/versions`,
		httpmock.NewStringResponse(http.StatusNotFound, `{"message":"404 Not Found"}`))

	output, err := runCommand(fakeHTTP, nil, false, "123")
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
		httpmock.NewStringResponse(http.StatusOK, `{
    "id": 123,
    "iid": 123,
    "project_id": 3,
    "title": "test1",
    "description": "fixed login page css paddings",
    "state": "merged"}`))

	testDiff := DiffTest(fakeHTTP)
	output, err := runCommand(fakeHTTP, nil, false, "")
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
		httpmock.NewStringResponse(http.StatusOK, `{
    "id": 123,
    "iid": 123,
    "project_id": 3,
    "title": "test1",
    "description": "fixed login page css paddings",
    "state": "merged"}`))

	DiffTest(fakeHTTP)
	output, err := runCommand(fakeHTTP, nil, true, "")
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
