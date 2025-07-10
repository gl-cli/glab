package update

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"gitlab.com/gitlab-org/cli/internal/glinstance"
	"gitlab.com/gitlab-org/cli/internal/testing/cmdtest"
	"gitlab.com/gitlab-org/cli/internal/testing/httpmock"
)

func TestUpdateCmd(t *testing.T) {
	type httpMock struct {
		method       string
		path         string
		requestBody  string
		status       int
		responseBody string
	}

	testCases := []struct {
		description string
		args        string
		httpMocks   []httpMock
		errString   string
	}{{
		description: "Update description for current repo",
		args:        "--description foo",
		httpMocks: []httpMock{{
			method:       http.MethodPut,
			path:         "https://gitlab.com/api/v4/projects/user%2Frepo",
			requestBody:  `{"description": "foo"}`,
			status:       http.StatusOK,
			responseBody: `{"name_with_namespace":"user / repo","web_url":"https://gitlab.com/user/repo"}`,
		}},
	}, {
		description: "Update description for user's repo",
		args:        "repo --description foo",
		httpMocks: []httpMock{{
			method:       http.MethodGet,
			path:         "https://gitlab.com/api/v4/user",
			status:       http.StatusOK,
			responseBody: `{ "username": "test_user" }`,
		}, {
			method:       http.MethodPut,
			path:         "https://gitlab.com/api/v4/projects/test_user%2Frepo",
			requestBody:  `{"description": "foo"}`,
			status:       http.StatusOK,
			responseBody: `{"name_with_namespace":"test_user / repo","web_url":"https://gitlab.com/test_user/repo"}`,
		}},
	}, {
		description: "Update description for other repo",
		args:        "otheruser/myproject --description foo",
		httpMocks: []httpMock{{
			method:       http.MethodPut,
			path:         "https://gitlab.com/api/v4/projects/otheruser/myproject",
			requestBody:  `{"description": "foo"}`,
			status:       http.StatusOK,
			responseBody: `{"name_with_namespace":"otheruser / myproject","web_url":"https://gitlab.com/otheruser/myproject"}`,
		}},
	}, {
		description: "Update description for repo at URL",
		args:        "https://gitlab.com/user/project --description foo",
		httpMocks: []httpMock{{
			method:       http.MethodPut,
			path:         "https://gitlab.com/api/v4/projects/user/project",
			requestBody:  `{"description": "foo"}`,
			status:       http.StatusOK,
			responseBody: `{"name_with_namespace":"user / project","web_url":"https://gitlab.com/user/project"}`,
		}},
	}, {
		description: "Update default branch",
		args:        "--defaultBranch main2",
		httpMocks: []httpMock{{
			method:       http.MethodPut,
			path:         "https://gitlab.com/api/v4/projects/user%2Frepo",
			requestBody:  `{"default_branch": "main2"}`,
			status:       http.StatusOK,
			responseBody: `{"name_with_namespace":"user / repo","web_url":"https://gitlab.com/user/repo"}`,
		}},
	}, {
		description: "Update both description and default branch at the same time",
		args:        "--description foo --defaultBranch main2",
		httpMocks: []httpMock{{
			method:       http.MethodPut,
			path:         "https://gitlab.com/api/v4/projects/user%2Frepo",
			requestBody:  `{"description": "foo", "default_branch": "main2"}`,
			status:       http.StatusOK,
			responseBody: `{"name_with_namespace":"user / repo","web_url":"https://gitlab.com/user/repo"}`,
		}},
	}, {
		description: "No flags provided",
		args:        "",
		errString:   "at least one of the flags in the group",
	}, {
		description: "Archive project with just --archive flag",
		args:        "--archive",
		httpMocks: []httpMock{{
			method:       http.MethodPost,
			path:         "https://gitlab.com/api/v4/projects/user%2Frepo/archive",
			status:       http.StatusOK,
			responseBody: `{"name_with_namespace":"user / repo","web_url":"https://gitlab.com/user/repo"}`,
		}},
	}, {
		description: "Archive project with --archive=true",
		args:        "--archive=true",
		httpMocks: []httpMock{{
			method:       http.MethodPost,
			path:         "https://gitlab.com/api/v4/projects/user%2Frepo/archive",
			status:       http.StatusOK,
			responseBody: `{"name_with_namespace":"user / repo","web_url":"https://gitlab.com/user/repo"}`,
		}},
	}, {
		description: "Unarchive project",
		args:        "--archive=false",
		httpMocks: []httpMock{{
			method:       http.MethodPost,
			path:         "https://gitlab.com/api/v4/projects/user%2Frepo/unarchive",
			status:       http.StatusOK,
			responseBody: `{"name_with_namespace":"user / repo","web_url":"https://gitlab.com/user/repo"}`,
		}},
	}, {
		description: "Archive project and change description at the same time",
		args:        "--archive=true --description=foobar",
		httpMocks: []httpMock{{
			method:       http.MethodPut,
			path:         "https://gitlab.com/api/v4/projects/user%2Frepo",
			requestBody:  `{"description": "foobar"}`,
			status:       http.StatusOK,
			responseBody: `{"name_with_namespace":"user / repo","web_url":"https://gitlab.com/user/repo"}`,
		}, {
			method:       http.MethodPost,
			path:         "https://gitlab.com/api/v4/projects/user%2Frepo/archive",
			status:       http.StatusOK,
			responseBody: `{"name_with_namespace":"user / repo","web_url":"https://gitlab.com/user/repo"}`,
		}},
	}, {
		description: "Unarchive project and change default branch at the same time",
		args:        "--archive=false --defaultBranch=main2",
		httpMocks: []httpMock{{
			method:       http.MethodPut,
			path:         "https://gitlab.com/api/v4/projects/user%2Frepo",
			requestBody:  `{"default_branch": "main2"}`,
			status:       http.StatusOK,
			responseBody: `{"name_with_namespace":"user / repo","web_url":"https://gitlab.com/user/repo"}`,
		}, {
			method:       http.MethodPost,
			path:         "https://gitlab.com/api/v4/projects/user%2Frepo/unarchive",
			status:       http.StatusOK,
			responseBody: `{"name_with_namespace":"user / repo","web_url":"https://gitlab.com/user/repo"}`,
		}},
	}}

	for _, test := range testCases {
		t.Run(test.description, func(t *testing.T) {
			fakeHTTP := &httpmock.Mocker{
				MatchURL: httpmock.FullURL,
			}
			defer fakeHTTP.Verify(t)

			for _, mock := range test.httpMocks {
				if mock.requestBody == "" {
					fakeHTTP.RegisterResponder(mock.method, mock.path, httpmock.NewStringResponse(mock.status, mock.responseBody))
				} else {
					fakeHTTP.RegisterResponderWithBody(mock.method, mock.path, mock.requestBody, httpmock.NewStringResponse(mock.status, mock.responseBody))
				}
			}

			ios, _, stdout, stderr := cmdtest.TestIOStreams(cmdtest.WithTestIOStreamsAsTTY(true))

			c := cmdtest.NewTestApiClient(t, &http.Client{Transport: fakeHTTP}, "", glinstance.DefaultHostname)
			factory := cmdtest.NewTestFactory(ios,
				cmdtest.WithBaseRepo("user", "repo"),
				cmdtest.WithApiClient(c),
				cmdtest.WithGitLabClient(c.Lab()),
			)

			cmd := NewCmdUpdate(factory)
			_, err := cmdtest.ExecuteCommand(cmd, test.args, stdout, stderr)

			if test.errString == "" {
				assert.NoErrorf(t, err, "unexpected error running command `project update %s`: %v", test.args, err)
			} else {
				assert.ErrorContainsf(t, err, test.errString, "")
			}
		})
	}
}
