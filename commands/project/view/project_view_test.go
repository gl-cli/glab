package view

import (
	"net/http"
	"testing"

	"github.com/MakeNowJust/heredoc"

	"github.com/stretchr/testify/assert"
	"gitlab.com/gitlab-org/cli/pkg/httpmock"

	"gitlab.com/gitlab-org/cli/commands/cmdtest"

	"gitlab.com/gitlab-org/cli/test"
)

func runCommand(rt http.RoundTripper, isTTY bool, cli string) (*test.CmdOut, error) {
	ios, _, stdout, stderr := cmdtest.InitIOStreams(isTTY, "")

	factory := cmdtest.InitFactory(ios, rt)

	factory.Branch = func() (string, error) {
		return "current-branch", nil
	}

	_, _ = factory.HttpClient()

	cmd := NewCmdView(factory)

	return cmdtest.ExecuteCommand(cmd, cli, stdout, stderr)
}

func TestProjectView(t *testing.T) {
	type httpMock struct {
		method string
		path   string
		status int
		body   string
	}

	tests := []struct {
		name      string
		cli       string
		httpMocks []httpMock

		expectedOutput string
	}{
		{
			name: "view the project details for the current project",
			cli:  "",
			httpMocks: []httpMock{
				{
					http.MethodGet,
					"/api/v4/projects/OWNER/REPO?license=true&statistics=true&with_custom_attributes=true",
					http.StatusOK,
					`{
							  "id": 37777023,
							  "description": "this is a test description",
							  "name": "REPO",
							  "name_with_namespace": "Test User / REPO",
							  "path": "REPO",
							  "path_with_namespace": "OWNER/REPO",
							  "created_at": "2022-07-13T02:04:56.151Z",
							  "default_branch": "main",
							  "http_url_to_repo": "https://gitlab.com/OWNER/REPO.git",
							  "web_url": "https://gitlab.com/OWNER/REPO",
							  "readme_url": "https://gitlab.com/OWNER/REPO/-/blob/main/README.md"
							}`,
				},
				{
					http.MethodGet,
					"/api/v4/projects/OWNER/REPO/repository/files/README%2Emd?ref=current-branch",
					http.StatusOK,
					`{"file_name": "README.md",
							  "file_path": "README.md",
							  "encoding": "base64",
							  "ref": "main",
							  "execute_filemode": false,
							  "content": "dGVzdCByZWFkbWUK"
							}`,
				},
			},
			expectedOutput: heredoc.Doc(`name:	Test User / REPO
												description:	this is a test description
												---
												test readme

										`),
		},
		{
			name: "view the details of a project owned by the current user",
			cli:  "foo",
			httpMocks: []httpMock{
				{
					http.MethodGet,
					"/api/v4/user",
					http.StatusOK,
					`{ "username": "test_user" }`,
				},
				{
					http.MethodGet,
					"/api/v4/projects/test_user/foo?license=true&statistics=true&with_custom_attributes=true",
					http.StatusOK,
					`{
							  "id": 37777023,
							  "description": "this is a test description",
							  "name": "foo",
							  "name_with_namespace": "test_user / foo",
							  "path": "foo",
							  "path_with_namespace": "test_user/foo",
							  "created_at": "2022-07-13T02:04:56.151Z",
							  "default_branch": "main",
							  "http_url_to_repo": "https://gitlab.com/test_user/foo.git",
							  "web_url": "https://gitlab.com/test_user/foo",
							  "readme_url": "https://gitlab.com/test_user/foo/-/blob/main/README.md"
							}`,
				},
				{
					http.MethodGet,
					"/api/v4/projects/test_user/foo/repository/files/README%2Emd?ref=main",
					http.StatusOK,
					`{"file_name": "README.md",
							  "file_path": "README.md",
							  "encoding": "base64",
							  "ref": "main",
							  "execute_filemode": false,
							  "content": "dGVzdCByZWFkbWUK"
							}`,
				},
			},
			expectedOutput: heredoc.Doc(`name:	test_user / foo
												description:	this is a test description
												---
												test readme

										`),
		},
		{
			name: "view a specific project's details",
			cli:  "foo/bar",
			httpMocks: []httpMock{
				{
					http.MethodGet,
					"/api/v4/projects/foo/bar?license=true&statistics=true&with_custom_attributes=true",
					http.StatusOK,
					`{
							  "id": 37777023,
							  "description": "this is a test description",
							  "name": "bar",
							  "name_with_namespace": "foo / bar",
							  "path": "bar",
							  "path_with_namespace": "foo/bar",
							  "created_at": "2022-07-13T02:04:56.151Z",
							  "default_branch": "main",
							  "http_url_to_repo": "https://gitlab.com/foo/bar.git",
							  "web_url": "https://gitlab.com/foo/bar",
							  "readme_url": "https://gitlab.com/foo/bar/-/blob/main/README.md"
							}`,
				},
				{
					http.MethodGet,
					"/api/v4/projects/foo/bar/repository/files/README%2Emd?ref=main",
					http.StatusOK,
					`{
							"file_name": "README.md",
							"file_path": "README.md",
							"encoding": "base64",
							"ref": "main",
							"execute_filemode": false,
							"content": "dGVzdCByZWFkbWUK"
							}`,
				},
			},
			expectedOutput: heredoc.Doc(`name:	foo / bar
												description:	this is a test description
												---
												test readme

										`),
		},
		{
			name: "view a group's specific project details",
			cli:  "group/foo/bar",
			httpMocks: []httpMock{
				{
					http.MethodGet,
					"/api/v4/projects/group/foo/bar?license=true&statistics=true&with_custom_attributes=true",
					http.StatusOK,
					`{
							  "id": 37777023,
							  "description": "this is a test description",
							  "name": "bar",
							  "name_with_namespace": "group / foo / bar",
							  "path": "bar",
							  "path_with_namespace": "group/foo/bar",
							  "created_at": "2022-07-13T02:04:56.151Z",
							  "default_branch": "main",
							  "http_url_to_repo": "https://gitlab.com/group/foo/bar.git",
							  "web_url": "https://gitlab.com/group/foo/bar",
							  "readme_url": "https://gitlab.com/group/foo/bar/-/blob/main/README.md"
							}`,
				},
				{
					http.MethodGet,
					"/api/v4/projects/group/foo/bar/repository/files/README%2Emd?ref=main",
					http.StatusOK,
					`{
							"file_name": "README.md",
							"file_path": "README.md",
							"encoding": "base64",
							"ref": "main",
							"execute_filemode": false,
							"content": "dGVzdCByZWFkbWUK"
							}`,
				},
			},
			expectedOutput: heredoc.Doc(`name:	group / foo / bar
												description:	this is a test description
												---
												test readme

										`),
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			fakeHTTP := &httpmock.Mocker{
				MatchURL: httpmock.PathAndQuerystring,
			}
			defer fakeHTTP.Verify(t)

			for _, mock := range tc.httpMocks {
				fakeHTTP.RegisterResponder(mock.method, mock.path, httpmock.NewStringResponse(mock.status, mock.body))
			}

			output, err := runCommand(fakeHTTP, false, tc.cli)

			if assert.NoErrorf(t, err, "error running command `project view %s`: %v", tc.cli, err) {
				assert.Equal(t, tc.expectedOutput, output.String())
				assert.Empty(t, output.Stderr())
			}
		})
	}
}
