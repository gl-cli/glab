package list

import (
	"net/http"
	"testing"

	"gitlab.com/gitlab-org/cli/internal/glinstance"
	"gitlab.com/gitlab-org/cli/internal/testing/cmdtest"

	"github.com/stretchr/testify/assert"
	"gitlab.com/gitlab-org/cli/internal/testing/httpmock"
	"gitlab.com/gitlab-org/cli/test"
)

func runCommand(t *testing.T, rt http.RoundTripper) (*test.CmdOut, error) {
	ios, _, stdout, stderr := cmdtest.TestIOStreams()
	factory := cmdtest.NewTestFactory(ios,
		cmdtest.WithApiClient(cmdtest.NewTestApiClient(t, &http.Client{Transport: rt}, "", glinstance.DefaultHostname)),
	)
	cmd := NewCmdList(factory)
	return cmdtest.ExecuteCommand(cmd, "", stdout, stderr)
}

func TestSSHKeyList(t *testing.T) {
	type httpMock struct {
		method string
		path   string
		status int
		body   string
	}
	keyResponse := `[{
    "id": 1,
    "title": "key title",
    "created_at": "2025-01-01T00:00:00.000Z",
    "expires_at": null,
    "key": "ssh-ed25519 example",
    "usage_type": "auth_and_signing"
  }]`

	tests := []struct {
		name        string
		httpMock    []httpMock
		expectedOut string
	}{
		{
			name: "when no ssh-keys are found shows an empty list",
			httpMock: []httpMock{{
				http.MethodGet,
				"/api/v4/user/keys?page=1&per_page=30",
				http.StatusOK,
				"[]",
			}},
			expectedOut: "\n",
		},
		{
			name: "when ssh-keys are found shows a list of keys",
			httpMock: []httpMock{{
				http.MethodGet,
				"/api/v4/user/keys?page=1&per_page=30",
				http.StatusOK,
				keyResponse,
			}},
			expectedOut: "Title\tKey\tUsage type\tCreated At\nkey title\tssh-ed25519 example\tauth_and_signing\t2025-01-01 00:00:00 +0000 UTC\n\n",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			fakeHTTP := &httpmock.Mocker{
				MatchURL: httpmock.PathAndQuerystring,
			}
			defer fakeHTTP.Verify(t)

			for _, mock := range tc.httpMock {
				fakeHTTP.RegisterResponder(mock.method, mock.path,
					httpmock.NewStringResponse(mock.status, mock.body))
			}

			output, err := runCommand(t, fakeHTTP)

			if assert.NoErrorf(t, err, "error running command `ssh-key list %s`: %v", err) {
				out := output.String()

				assert.Equal(t, tc.expectedOut, out)
				assert.Empty(t, output.Stderr())
			}
		})
	}
}
