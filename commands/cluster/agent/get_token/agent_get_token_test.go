package get_token

import (
	"net/http"
	"testing"

	"github.com/MakeNowJust/heredoc/v2"
	"github.com/stretchr/testify/assert"
	"gitlab.com/gitlab-org/cli/commands/cmdtest"
	"gitlab.com/gitlab-org/cli/pkg/httpmock"
	"gitlab.com/gitlab-org/cli/test"
)

func runCommand(rt http.RoundTripper, isTTY bool, cli string, doHyperlinks string) (*test.CmdOut, error) {
	ios, _, stdout, stderr := cmdtest.InitIOStreams(isTTY, doHyperlinks)
	f := cmdtest.InitFactory(ios, rt)

	// Note: This sets the RoundTripper, which is necessary for stubs to work.
	_, _ = f.HttpClient()

	cmd := NewCmdAgentGetToken(f)

	return cmdtest.ExecuteCommand(cmd, cli, stdout, stderr)
}

func TestAgentGetToken(t *testing.T) {
	fakeHTTP := httpmock.New()
	defer fakeHTTP.Verify(t)

	fakeHTTP.RegisterResponder(http.MethodPost, "/user/personal_access_tokens",
		httpmock.NewStringResponse(http.StatusOK, `
		  {
			"token": "glpat-XTESTX",
			"expires_at": "2023-01-02"
		  }
		`))

	output, err := runCommand(fakeHTTP, true, "--agent 42", "")
	if err != nil {
		t.Errorf("error running command `cluster agent get-token --agent 42`: %v", err)
	}

	assert.Equal(t, heredoc.Doc(`
		{
		  "kind": "ExecCredential",
		  "apiVersion": "client.authentication.k8s.io/v1",
		  "spec": {
		    "interactive": false
		  },
		  "status": {
		    "expirationTimestamp": "2023-01-01T23:55:00Z",
		    "token": "pat:42:glpat-XTESTX"
		  }
		}
	`), output.String())
	assert.Equal(t, ``, output.Stderr())
}
