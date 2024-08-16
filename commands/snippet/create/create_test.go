package create

import (
	"errors"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
	"gitlab.com/gitlab-org/cli/commands/cmdtest"
	"gitlab.com/gitlab-org/cli/pkg/httpmock"
	"gitlab.com/gitlab-org/cli/test"
)

func TestSnippetCreate(t *testing.T) {
	type httpMock struct {
		method string
		path   string
		status int
		body   string
	}

	testCases := []struct {
		name       string
		command    string
		wantErr    error
		wantStderr []string
		wantStdout []string
		mock       httpMock
	}{
		{
			name:       "Create personal snippet",
			command:    "testdata/snippet.txt --personal -d 'Hello World snippet' -f 'snippet.txt' -t 'This is a snippet'",
			wantStderr: []string{"- Creating snippet in personal space"},
			wantStdout: []string{"https://gitlab.example.com/-/snippets/1"},
			mock: httpMock{
				method: http.MethodPost,
				path:   "/api/v4/snippets",
				status: http.StatusCreated,
				body: `{
  "id": 1,
  "title": "This is a snippet",
  "description": "Hello World snippet",
  "web_url": "https://gitlab.example.com/-/snippets/1",
  "file_name": "snippet.txt",
  "files": [
    {
      "path": "snippet.txt",
      "raw_url": "https://gitlab.example.com/-/snippets/1/raw/main/snippet.txt"
    }
  ]
}`,
			},
		},
		{
			name:       "Create project snippet",
			command:    "testdata/snippet.txt -d 'Hello World snippet' -f 'snippet.txt' -t 'This is a snippet'",
			wantStderr: []string{"- Creating snippet in OWNER/REPO"},
			wantStdout: []string{"https://gitlab.example.com/OWNER/REPO/-/snippets/1"},
			mock: httpMock{
				method: http.MethodPost,
				path:   "/api/v4/projects/OWNER/REPO/snippets",
				status: http.StatusCreated,
				body: `{
  "id": 1,
  "title": "This is a snippet",
  "description": "Hello World snippet",
  "web_url": "https://gitlab.example.com/OWNER/REPO/-/snippets/1",
  "file_name": "snippet.txt",
  "files": [
    {
      "path": "snippet.txt",
      "raw_url": "https://gitlab.example.com/-/OWNER/REPO/snippets/1/raw/main/snippet.txt"
    }
  ]
}`,
			},
		},

		{
			name:       "Create project snippet using a path",
			command:    "testdata/snippet.txt -d 'Hello World snippet' -t 'This is a snippet'",
			wantStderr: []string{"- Creating snippet in OWNER/REPO"},
			wantStdout: []string{"https://gitlab.example.com/OWNER/REPO/-/snippets/1"},
			mock: httpMock{
				method: http.MethodPost,
				path:   "/api/v4/projects/OWNER/REPO/snippets",
				status: http.StatusCreated,
				body: `{
  "id": 1,
  "title": "This is a snippet",
  "description": "Hello World snippet",
  "web_url": "https://gitlab.example.com/OWNER/REPO/-/snippets/1",
  "file_name": "test.txt",
  "files": [
    {
      "path": "text.txt",
      "raw_url": "https://gitlab.example.com/-/OWNER/REPO/snippets/1/raw/main/text.txt"
    }
  ]
}`,
			},
		},
		{
			name:    "Create snippet 403 failure",
			command: "testdata/snippet.txt -d 'Hello World snippet' -f 'snippet.txt' -t 'This is a snippet'",
			wantErr: errors.New("failed to create snippet: POST https://gitlab.com/api/v4/projects/OWNER/REPO/snippets: 403"),
			mock: httpMock{
				method: http.MethodPost,
				path:   "/api/v4/projects/OWNER/REPO/snippets",
				status: http.StatusForbidden,
				body:   "",
			},
		},
		{
			name:    "Create personal snippet 403 failure",
			command: "testdata/snippet.txt --personal -d 'Hello World snippet' -f 'snippet.txt' -t 'This is a personal snippet'",
			wantErr: errors.New("failed to create snippet: POST https://gitlab.com/api/v4/snippets: 403"),
			mock: httpMock{
				method: http.MethodPost,
				path:   "/api/v4/snippets",
				status: http.StatusForbidden,
				body:   "",
			},
		},
		{
			name:    "Create snippet no stdin failure",
			command: "-d 'Hello World snippet' -f 'snippet.txt' -t 'This is a personal snippet'",
			wantErr: errors.New("stdin required if no 'path' is provided"),
		},
		{
			name:    "Create snippet no path failure",
			command: "-d 'Hello World snippet' -t 'This is a personal snippet'",
			wantErr: errors.New("if 'path' is not provided, 'filename' and stdin are required"),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			fakeHTTP := &httpmock.Mocker{
				MatchURL: httpmock.PathOnly,
			}
			defer fakeHTTP.Verify(t)

			if tc.mock.method != "" || tc.mock.path != "" {
				fakeHTTP.RegisterResponder(tc.mock.method, tc.mock.path, httpmock.NewStringResponse(tc.mock.status, tc.mock.body))
			}

			out, err := runCommand(fakeHTTP, tc.command)
			if tc.wantErr != nil {
				require.EqualError(t, err, tc.wantErr.Error())
			} else {
				require.NoError(t, err)
			}

			for _, msg := range tc.wantStdout {
				require.Contains(t, out.String(), msg)
			}

			for _, msg := range tc.wantStderr {
				require.Contains(t, out.Stderr(), msg)
			}
		})
	}
}

func runCommand(rt http.RoundTripper, cli string) (*test.CmdOut, error) {
	ios, _, stdout, stderr := cmdtest.InitIOStreams(false, "")
	factory := cmdtest.InitFactory(ios, rt)
	_, _ = factory.HttpClient()
	cmd := NewCmdCreate(factory)
	return cmdtest.ExecuteCommand(cmd, cli, stdout, stderr)
}
