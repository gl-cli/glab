package list

import (
	"fmt"
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
		cmdtest.WithGitLabClient(cmdtest.NewTestApiClient(t, &http.Client{Transport: rt}, "", glinstance.DefaultHostname).Lab()),
	)
	cmd := NewCmdList(factory)
	return cmdtest.ExecuteCommand(cmd, "", stdout, stderr)
}

func TestDeployKeyList(t *testing.T) {
	type httpMock struct {
		method string
		path   string
		status int
		body   string
	}
	keyResponse := `[
  {
    "id": 1,
    "title": "example key",
    "key": "ssh-ed25519 example",
    "fingerprint": "1a:2b:3c:4d:5e:6f:7g:8h:9i:0j:kl:mn:op:qr:st:uv:wx:yz:1a:",
    "fingerprint_sha256": "SHA256:example",
    "created_at": "2025-01-01T00:00:00Z",
    "expires_at": null,
    "can_push": false
  }]`

	repoName := "OWNER%2FREPO"
	pagination := "?page=1&per_page=30"
	apiEndpoint := fmt.Sprintf("/api/v4/projects/%s/deploy_keys%s", repoName, pagination)

	tests := []struct {
		name        string
		httpMock    []httpMock
		expectedOut string
	}{
		{
			name: "when no deploy keys are found shows an empty list",
			httpMock: []httpMock{{
				http.MethodGet,
				apiEndpoint,
				http.StatusOK,
				"[]",
			}},
			expectedOut: "\n",
		},
		{
			name: "when deploy keys are found shows a list of keys",
			httpMock: []httpMock{{
				http.MethodGet,
				apiEndpoint,
				http.StatusOK,
				keyResponse,
			}},
			expectedOut: "Title\tKey\tCan Push\tCreated At\nexample key\tssh-ed25519 example\tfalse\t2025-01-01 00:00:00 +0000 UTC\n\n",
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

			if assert.NoErrorf(t, err, "error running command `deploy-key list %s`: %v", err) {
				out := output.String()

				assert.Equal(t, tc.expectedOut, out)
				assert.Empty(t, output.Stderr())
			}
		})
	}
}
