package git

import (
	"os"
	"os/exec"
	"testing"

	"gitlab.com/gitlab-org/cli/internal/run"

	"github.com/stretchr/testify/require"
)

func InitGitRepo(t *testing.T) string {
	tempDir := t.TempDir()

	err := os.Chdir(tempDir)
	require.NoError(t, err)

	gitInit := GitCommand("init")
	_, err = run.PrepareCmd(gitInit).Output()
	require.NoError(t, err)

	return tempDir
}

func InitGitRepoWithCommit(t *testing.T) string {
	tempDir := InitGitRepo(t)

	configureGitConfig(t)

	err := exec.Command("touch", "randomfile").Run()
	require.NoError(t, err)

	gitAdd := GitCommand("add", "randomfile")
	_, err = run.PrepareCmd(gitAdd).Output()
	require.NoError(t, err)

	gitCommit := GitCommand("commit", "-m", "\"commit\"")
	_, err = run.PrepareCmd(gitCommit).Output()
	require.NoError(t, err)

	return tempDir
}

func configureGitConfig(t *testing.T) {
	// CI will throw errors using a git command without a configuration
	nameConfig := GitCommand("config", "user.name", "glab test bot")
	_, err := run.PrepareCmd(nameConfig).Output()
	require.NoError(t, err)

	emailConfig := GitCommand("config", "user.email", "no-reply+cli-tests@gitlab.com")
	_, err = run.PrepareCmd(emailConfig).Output()
	require.NoError(t, err)
}

func CreateRefFiles(refs map[string]StackRef, title string) error {
	for _, ref := range refs {
		err := AddStackRefFile(title, ref)
		if err != nil {
			return err
		}
	}

	return nil
}

func CreateBranches(t *testing.T, branches []string) {
	// older versions of git could default to a different branch,
	// so making sure this one exists.
	_ = CheckoutNewBranch("main")

	for _, branch := range branches {
		err := CheckoutNewBranch(branch)
		require.Nil(t, err)
	}
}
