//go:build !integration

package init

import (
	"io"
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	gitlab "gitlab.com/gitlab-org/api/client-go"
	gitlabtesting "gitlab.com/gitlab-org/api/client-go/testing"

	"gitlab.com/gitlab-org/cli/internal/api"
	"gitlab.com/gitlab-org/cli/internal/cmdutils"
	"gitlab.com/gitlab-org/cli/internal/testing/cmdtest"
)

func TestInit_CommandConstruction(t *testing.T) {
	// GIVEN
	tc := gitlabtesting.NewTestClient(t, gitlab.WithBaseURL("https://gitlab.example.com"))
	exec := cmdtest.SetupCmdForTest(
		t,
		func(f cmdutils.Factory) *cobra.Command {
			// THEN
			return NewCmd(f, runCommandMatcher(t, "my-tofu", []string{
				"-chdir=infra",
				"init",
				"-backend-config=address=https://gitlab.example.com/api/v4/projects/OWNER%2FREPO/terraform/state/production",
				"-backend-config=lock_address=https://gitlab.example.com/api/v4/projects/OWNER%2FREPO/terraform/state/production/lock",
				"-backend-config=unlock_address=https://gitlab.example.com/api/v4/projects/OWNER%2FREPO/terraform/state/production/lock",
				"-backend-config=lock_method=POST",
				"-backend-config=unlock_method=DELETE",
				"-backend-config=retry_wait_min=5",
				"-backend-config=headers={\"Authorization\" = \"Bearer testtoken\"}",
			}))
		},
		false,
		cmdtest.WithApiClient(cmdtest.NewTestApiClient(t, nil, "testtoken", "gitlab.example.com", api.WithGitLabClient(tc.Client))),
		cmdtest.WithBaseRepo("OWNER", "REPO", "gitlab.example.com"),
	)

	// WHEN
	_, err := exec("production -d infra -b my-tofu")
	require.NoError(t, err)
}

func TestInit_CommandConstruction_InitArgs(t *testing.T) {
	// GIVEN
	tc := gitlabtesting.NewTestClient(t, gitlab.WithBaseURL("https://gitlab.example.com"))
	exec := cmdtest.SetupCmdForTest(
		t,
		func(f cmdutils.Factory) *cobra.Command {
			// THEN
			return NewCmd(f, runCommandMatcher(t, "my-tofu", []string{
				"-chdir=infra",
				"init",
				"-backend-config=address=https://gitlab.example.com/api/v4/projects/OWNER%2FREPO/terraform/state/production",
				"-backend-config=lock_address=https://gitlab.example.com/api/v4/projects/OWNER%2FREPO/terraform/state/production/lock",
				"-backend-config=unlock_address=https://gitlab.example.com/api/v4/projects/OWNER%2FREPO/terraform/state/production/lock",
				"-backend-config=lock_method=POST",
				"-backend-config=unlock_method=DELETE",
				"-backend-config=retry_wait_min=5",
				"-backend-config=headers={\"Authorization\" = \"Bearer testtoken\"}",
				"-reconfigure",
			}))
		},
		false,
		cmdtest.WithApiClient(cmdtest.NewTestApiClient(t, nil, "testtoken", "gitlab.example.com", api.WithGitLabClient(tc.Client))),
		cmdtest.WithBaseRepo("OWNER", "REPO", "gitlab.example.com"),
	)

	// WHEN
	_, err := exec("production -d infra -b my-tofu -- -reconfigure")
	require.NoError(t, err)
}

func runCommandMatcher(t *testing.T, expectedBinary string, expectedArgs []string) RunCommandFunc {
	t.Helper()

	return func(stdout, stderr io.Writer, binary string, args []string) error {
		assert.Equal(t, expectedBinary, binary)
		assert.Equal(t, expectedArgs, args)
		return nil
	}
}
