package view

import (
	"net/http"
	"os/exec"
	"testing"

	"gitlab.com/gitlab-org/cli/internal/glrepo"

	"github.com/MakeNowJust/heredoc/v2"

	"github.com/stretchr/testify/assert"
	"gitlab.com/gitlab-org/cli/pkg/httpmock"

	"gitlab.com/gitlab-org/cli/commands/cmdtest"
	"gitlab.com/gitlab-org/cli/internal/run"
	"gitlab.com/gitlab-org/cli/test"
)

func runCommand(rt http.RoundTripper, isTTY bool, cli string, stub bool, repoHost string) (*test.CmdOut, error, func()) {
	ios, _, stdout, stderr := cmdtest.InitIOStreams(isTTY, "")

	factory := cmdtest.InitFactory(ios, rt)

	factory.Branch = func() (string, error) {
		return "#current-branch", nil
	}

	factory.BaseRepo = func() (glrepo.Interface, error) {
		if repoHost == "" {
			return glrepo.New("OWNER", "REPO"), nil
		} else {
			return glrepo.NewWithHost("OWNER", "REPO", repoHost), nil
		}
	}

	_, _ = factory.HttpClient()

	cmd := NewCmdView(factory)

	var restoreCmd func()

	if stub {
		restoreCmd = run.SetPrepareCmd(func(cmd *exec.Cmd) run.Runnable {
			return &test.OutputStub{}
		})
	}

	cmdOut, err := cmdtest.ExecuteCommand(cmd, cli, stdout, stderr)

	return cmdOut, err, restoreCmd
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
		isTTY     bool
		stub      bool
		repoHost  string

		expectedOutput string
	}{
		{
			name: "view the project details for the current project",
			cli:  "",
			httpMocks: []httpMock{
				{
					http.MethodGet,
					"https://gitlab.com/api/v4/projects/OWNER%2FREPO?license=true&with_custom_attributes=true",
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
					"https://gitlab.com/api/v4/projects/OWNER%2FREPO/repository/files/README%2Emd?ref=%23current-branch",
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
					"https://gitlab.com/api/v4/user",
					http.StatusOK,
					`{ "username": "test_user" }`,
				},
				{
					http.MethodGet,
					"https://gitlab.com/api/v4/projects/test_user%2Ffoo?license=true&with_custom_attributes=true",
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
					"https://gitlab.com/api/v4/projects/test_user%2Ffoo/repository/files/README%2Emd?ref=main",
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
					"https://gitlab.com/api/v4/projects/foo%2Fbar?license=true&with_custom_attributes=true",
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
					"https://gitlab.com/api/v4/projects/foo%2Fbar/repository/files/README%2Emd?ref=main",
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
					"https://gitlab.com/api/v4/projects/group%2Ffoo%2Fbar?license=true&with_custom_attributes=true",
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
					"https://gitlab.com/api/v4/projects/group%2Ffoo%2Fbar/repository/files/README%2Emd?ref=main",
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
		{
			name: "view a project details from a project not hosted on the default host",
			cli:  "",
			httpMocks: []httpMock{
				{
					http.MethodGet,
					"https://gitlab.company.org/api/v4/projects/OWNER%2FREPO?license=true&with_custom_attributes=true",
					http.StatusOK,
					`{
							  "id": 37777023,
							  "description": "this is a test description",
							  "name": "bar",
							  "name_with_namespace": "OWNER / REPO",
							  "path": "bar",
							  "path_with_namespace": "OWNER/REPO",
							  "created_at": "2022-07-13T02:04:56.151Z",
							  "default_branch": "main",
							  "http_url_to_repo": "https://gitlab.company.org/OWNER/REPO.git",
							  "web_url": "https://gitlab.company.org/OWNER/REPO",
							  "readme_url": "https://gitlab.company.org/OWNER/REPO/-/blob/main/README.md"
							}`,
				},
				{
					http.MethodGet,
					"https://gitlab.company.org/api/v4/projects/OWNER%2FREPO/repository/files/README%2Emd?ref=%23current-branch",
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
			repoHost: "gitlab.company.org",
			expectedOutput: heredoc.Doc(`name:	OWNER / REPO
												description:	this is a test description
												---
												test readme

										`),
		},
		{
			name: "view project details from a git URL",
			cli:  "https://gitlab.company.org/OWNER/REPO.git",
			httpMocks: []httpMock{
				{
					http.MethodGet,
					"https://gitlab.company.org/api/v4/projects/OWNER%2FREPO?license=true&with_custom_attributes=true",
					http.StatusOK,
					`{
							  "id": 37777023,
							  "description": "this is a test description",
							  "name": "bar",
							  "name_with_namespace": "OWNER / REPO",
							  "path": "bar",
							  "path_with_namespace": "OWNER/REPO",
							  "created_at": "2022-07-13T02:04:56.151Z",
							  "default_branch": "main",
							  "http_url_to_repo": "https://gitlab.company.org/OWNER/REPO.git",
							  "web_url": "https://gitlab.company.org/OWNER/REPO",
							  "readme_url": "https://gitlab.company.org/OWNER/REPO/-/blob/main/README.md"
							}`,
				},
				{
					http.MethodGet,
					"https://gitlab.company.org/api/v4/projects/OWNER%2FREPO/repository/files/README%2Emd?ref=main",
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
			expectedOutput: heredoc.Doc(`name:	OWNER / REPO
												description:	this is a test description
												---
												test readme

										`),
		},
		{
			name: "view project on web where current branch is different to default branch",
			cli:  "--web",
			httpMocks: []httpMock{
				{
					http.MethodGet,
					"https://gitlab.com/api/v4/projects/OWNER%2FREPO?license=true&with_custom_attributes=true",
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
			},
			isTTY:          true,
			stub:           true,
			expectedOutput: "Opening gitlab.com/OWNER/REPO/-/tree/%23current-branch in your browser.\n",
		},
		{
			name: "view project default branch on web",
			cli:  "--web",
			httpMocks: []httpMock{
				{
					http.MethodGet,
					"https://gitlab.com/api/v4/projects/OWNER%2FREPO?license=true&with_custom_attributes=true",
					http.StatusOK,
					`{
							  "id": 37777023,
							  "description": "this is a test description",
							  "name": "REPO",
							  "name_with_namespace": "Test User / REPO",
							  "path": "REPO",
							  "path_with_namespace": "OWNER/REPO",
							  "created_at": "2022-07-13T02:04:56.151Z",
							  "default_branch": "#current-branch",
							  "http_url_to_repo": "https://gitlab.com/OWNER/REPO.git",
							  "web_url": "https://gitlab.com/OWNER/REPO",
							  "readme_url": "https://gitlab.com/OWNER/REPO/-/blob/main/README.md"
							}`,
				},
			},
			isTTY:          true,
			stub:           true,
			expectedOutput: "Opening gitlab.com/OWNER/REPO in your browser.\n",
		},
		{
			name: "view project when passing a https git URL on web",
			cli:  "https://gitlab.company.org/OWNER/REPO.git --web",
			httpMocks: []httpMock{
				{
					http.MethodGet,
					"https://gitlab.company.org/api/v4/projects/OWNER%2FREPO?license=true&with_custom_attributes=true",
					http.StatusOK,
					`{
							  "id": 37777023,
							  "description": "this is a test description",
							  "name": "REPO",
							  "name_with_namespace": "Test User / REPO",
							  "path": "REPO",
							  "path_with_namespace": "OWNER/REPO",
							  "created_at": "2022-07-13T02:04:56.151Z",
							  "default_branch": "#current-branch",
							  "http_url_to_repo": "https://gitlab.company.org/OWNER/REPO.git",
							  "web_url": "https://gitlab.company.org/OWNER/REPO",
							  "readme_url": "https://gitlab.company.org/OWNER/REPO/-/blob/main/README.md"
							}`,
				},
			},
			isTTY:          true,
			stub:           true,
			expectedOutput: "Opening gitlab.company.org/OWNER/REPO in your browser.\n",
		},
		{
			name: "view project when passing a https git URL on web",
			cli:  "https://gitlab.company.org/OWNER/REPO.git --web --branch foobranch",
			httpMocks: []httpMock{
				{
					http.MethodGet,
					"https://gitlab.company.org/api/v4/projects/OWNER%2FREPO?license=true&with_custom_attributes=true",
					http.StatusOK,
					`{
							  "id": 37777023,
							  "description": "this is a test description",
							  "name": "REPO",
							  "name_with_namespace": "Test User / REPO",
							  "path": "REPO",
							  "path_with_namespace": "OWNER/REPO",
							  "created_at": "2022-07-13T02:04:56.151Z",
							  "default_branch": "#current-branch",
							  "http_url_to_repo": "https://gitlab.company.org/OWNER/REPO.git",
							  "web_url": "https://gitlab.company.org/OWNER/REPO",
							  "readme_url": "https://gitlab.company.org/OWNER/REPO/-/blob/main/README.md"
							}`,
				},
			},
			isTTY:          true,
			stub:           true,
			expectedOutput: "Opening gitlab.company.org/OWNER/REPO/-/tree/foobranch in your browser.\n",
		},
		{
			name: "view project when passing a https URL on web",
			cli:  "https://gitlab.company.org/OWNER/REPO --web",
			httpMocks: []httpMock{
				{
					http.MethodGet,
					"https://gitlab.company.org/api/v4/projects/OWNER%2FREPO?license=true&with_custom_attributes=true",
					http.StatusOK,
					`{
							  "id": 37777023,
							  "description": "this is a test description",
							  "name": "REPO",
							  "name_with_namespace": "Test User / REPO",
							  "path": "REPO",
							  "path_with_namespace": "OWNER/REPO",
							  "created_at": "2022-07-13T02:04:56.151Z",
							  "default_branch": "#current-branch",
							  "http_url_to_repo": "https://gitlab.company.org/OWNER/REPO.git",
							  "web_url": "https://gitlab.company.org/OWNER/REPO",
							  "readme_url": "https://gitlab.company.org/OWNER/REPO/-/blob/main/README.md"
							}`,
				},
			},
			isTTY:          true,
			stub:           true,
			expectedOutput: "Opening gitlab.company.org/OWNER/REPO in your browser.\n",
		},
		{
			name: "view project when passing a git URL on web",
			cli:  "git@gitlab.company.org:OWNER/REPO.git --web",
			httpMocks: []httpMock{
				{
					http.MethodGet,
					"https://gitlab.company.org/api/v4/projects/OWNER%2FREPO?license=true&with_custom_attributes=true",
					http.StatusOK,
					`{
							  "id": 37777023,
							  "description": "this is a test description",
							  "name": "REPO",
							  "name_with_namespace": "Test User / REPO",
							  "path": "REPO",
							  "path_with_namespace": "OWNER/REPO",
							  "created_at": "2022-07-13T02:04:56.151Z",
							  "default_branch": "#current-branch",
							  "http_url_to_repo": "https://gitlab.company.org/OWNER/REPO.git",
							  "web_url": "https://gitlab.company.org/OWNER/REPO",
							  "readme_url": "https://gitlab.company.org/OWNER/REPO/-/blob/main/README.md"
							}`,
				},
			},
			isTTY:          true,
			stub:           true,
			expectedOutput: "Opening gitlab.company.org/OWNER/REPO in your browser.\n",
		},
		{
			name: "view a project that isn't on the default host on web",
			cli:  "--web",
			httpMocks: []httpMock{
				{
					http.MethodGet,
					"https://gitlab.company.org/api/v4/projects/OWNER%2FREPO?license=true&with_custom_attributes=true",
					http.StatusOK,
					`{
							  "id": 37777023,
							  "description": "this is a test description",
							  "name": "REPO",
							  "name_with_namespace": "Test User / REPO",
							  "path": "REPO",
							  "path_with_namespace": "OWNER/REPO",
							  "created_at": "2022-07-13T02:04:56.151Z",
							  "default_branch": "#current-branch",
							  "http_url_to_repo": "https://gitlab.company.org/OWNER/REPO.git",
							  "web_url": "https://gitlab.company.org/OWNER/REPO",
							  "readme_url": "https://gitlab.company.org/OWNER/REPO/-/blob/main/README.md"
							}`,
				},
			},
			isTTY:          true,
			stub:           true,
			repoHost:       "gitlab.company.org",
			expectedOutput: "Opening gitlab.company.org/OWNER/REPO in your browser.\n",
		},
		{
			name: "view a specific project branch on the web",
			cli:  "--branch foo --web",
			httpMocks: []httpMock{
				{
					http.MethodGet,
					"https://gitlab.com/api/v4/projects/OWNER%2FREPO?license=true&with_custom_attributes=true",
					http.StatusOK,
					`{
							  "id": 37777023,
							  "description": "this is a test description",
							  "name": "REPO",
							  "name_with_namespace": "Test User / REPO",
							  "path": "REPO",
							  "path_with_namespace": "OWNER/REPO",
							  "created_at": "2022-07-13T02:04:56.151Z",
							  "default_branch": "#current-branch",
							  "http_url_to_repo": "https://gitlab.com/OWNER/REPO.git",
							  "web_url": "https://gitlab.com/OWNER/REPO",
							  "readme_url": "https://gitlab.com/OWNER/REPO/-/blob/main/README.md"
							}`,
				},
			},
			isTTY:          true,
			stub:           true,
			expectedOutput: "Opening gitlab.com/OWNER/REPO/-/tree/foo in your browser.\n",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			fakeHTTP := &httpmock.Mocker{
				MatchURL: httpmock.FullURL,
			}
			defer fakeHTTP.Verify(t)

			for _, mock := range tc.httpMocks {
				fakeHTTP.RegisterResponder(mock.method, mock.path, httpmock.NewStringResponse(mock.status, mock.body))
			}

			output, err, restoreCmd := runCommand(fakeHTTP, tc.isTTY, tc.cli, tc.stub, tc.repoHost)
			if restoreCmd != nil {
				defer restoreCmd()
			}

			if assert.NoErrorf(t, err, "error running command `project view %s`: %v", tc.cli, err) {
				assert.Equal(t, tc.expectedOutput, output.String())
				assert.Empty(t, output.Stderr())
			}
		})
	}
}
