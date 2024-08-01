package generate

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"gitlab.com/gitlab-org/cli/commands/cmdtest"
	"gitlab.com/gitlab-org/cli/pkg/httpmock"
	"gitlab.com/gitlab-org/cli/test"
)

func runCommand(cli string, rt http.RoundTripper) (*test.CmdOut, error) {
	ios, _, stdout, stderr := cmdtest.InitIOStreams(false, "")
	factory := cmdtest.InitFactory(ios, rt)

	_, _ = factory.HttpClient()
	cmd := NewCmdGenerate(factory)

	return cmdtest.ExecuteCommand(cmd, cli, stdout, stderr)
}

func TestChangelogGenerate(t *testing.T) {
	fakeHTTP := &httpmock.Mocker{MatchURL: httpmock.PathAndQuerystring}
	defer fakeHTTP.Verify(t)

	// Mock the project ID
	fakeHTTP.RegisterResponder(http.MethodGet, "/api/v4/projects/OWNER/REPO?license=true&with_custom_attributes=true",
		httpmock.NewStringResponse(http.StatusOK, `{ "id": 37777023 }`))

	// Mock the acutal changelog API call
	// TODO: mock the other optional attributes that we can pass to the endpoint.
	fakeHTTP.RegisterResponder(http.MethodGet, "/api/v4/projects/37777023/repository/changelog?version=1.0.0",
		httpmock.NewStringResponse(http.StatusOK, `{
			"notes": "## 1.0.0 (2023-04-02)\n\n### FirstName LastName firstname@lastname.com (1 changes)\n\n- [inital commit](gitlab-org/cli@somehash ([merge request](gitlab-org/cli!1))\n"
		}`))

	output, err := runCommand("--version 1.0.0", fakeHTTP)
	require.Nil(t, err)

	assert.Empty(t, output.Stderr())

	expectedStr := "## 1.0.0 (2023-04-02)\n\n### FirstName LastName firstname@lastname.com (1 changes)\n\n- [inital commit](gitlab-org/cli@somehash ([merge request](gitlab-org/cli!1))\n"
	assert.Equal(t, expectedStr, output.String())
}

func TestChangelogGenerateWithError(t *testing.T) {
	cases := map[string]struct {
		httpStatus  int
		httpMsgJSON string
		errorMsg    string
	}{
		"unauthorized": {
			httpStatus:  http.StatusUnauthorized,
			httpMsgJSON: "{message: 401 Unauthorized}",
			errorMsg:    "GET https://gitlab.com/api/v4/projects/37777023/repository/changelog: 401 failed to parse unknown error format: {message: 401 Unauthorized}",
		},
		"not found": {
			httpStatus:  http.StatusNotFound,
			httpMsgJSON: "{message: 404 Project Not Found}",
			errorMsg:    "404 Not Found",
		},
	}

	for name, v := range cases {
		t.Run(name, func(t *testing.T) {
			fakeHTTP := &httpmock.Mocker{MatchURL: httpmock.PathAndQuerystring}
			defer fakeHTTP.Verify(t)

			// Mock the project ID
			fakeHTTP.RegisterResponder(http.MethodGet, "/api/v4/projects/OWNER/REPO?license=true&with_custom_attributes=true",
				httpmock.NewStringResponse(http.StatusOK, `{ "id": 37777023 }`))

			fakeHTTP.RegisterResponder(http.MethodGet, "/api/v4/projects/37777023/repository/changelog?version=1.0.0",
				httpmock.NewStringResponse(v.httpStatus, v.httpMsgJSON))

			_, err := runCommand("--version 1.0.0", fakeHTTP)

			require.NotNil(t, err)
			require.Equal(t, v.errorMsg, err.Error())
		})
	}
}
