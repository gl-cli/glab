package get_token

import (
	"testing"
	"time"

	"github.com/MakeNowJust/heredoc/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	gitlab "gitlab.com/gitlab-org/api/client-go"
	gitlab_testing "gitlab.com/gitlab-org/api/client-go/testing"
	"gitlab.com/gitlab-org/cli/internal/testing/cmdtest"
	"go.uber.org/mock/gomock"
)

func TestAgentGetToken(t *testing.T) {
	// GIVEN
	tc := gitlab_testing.NewTestClient(t)
	exec := cmdtest.SetupCmdForTest(t, NewCmdAgentGetToken, cmdtest.WithGitLabClient(tc.Client))

	tc.MockUsers.EXPECT().
		CreatePersonalAccessTokenForCurrentUser(gomock.Any()).
		Return(&gitlab.PersonalAccessToken{
			Token:     "glpat-XTESTX",
			ExpiresAt: gitlab.Ptr(mustParse(t, "2023-01-02")),
		}, &gitlab.Response{}, nil).
		Times(1)

	// WHEN
	output, err := exec("--agent 42")
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

func mustParse(t *testing.T, dt string) gitlab.ISOTime {
	x, err := time.Parse(time.DateOnly, dt)
	require.NoError(t, err)
	return gitlab.ISOTime(x)
}
