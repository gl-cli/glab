package create

import (
	"path"
	"testing"

	"github.com/stretchr/testify/require"
	"gitlab.com/gitlab-org/cli/internal/config"
	"gitlab.com/gitlab-org/cli/internal/git"
	git_testing "gitlab.com/gitlab-org/cli/internal/git/testing"
	"gitlab.com/gitlab-org/cli/internal/glinstance"
	"gitlab.com/gitlab-org/cli/internal/prompt"
	"gitlab.com/gitlab-org/cli/internal/testing/cmdtest"
	"gitlab.com/gitlab-org/cli/test"
	"go.uber.org/mock/gomock"
)

func runCommand(t *testing.T, mockCmd git.GitRunner, args string) (*test.CmdOut, error) {
	ios, _, stdout, stderr := cmdtest.TestIOStreams(cmdtest.WithTestIOStreamsAsTTY(true))
	factory := cmdtest.NewTestFactory(ios,
		cmdtest.WithGitLabClient(cmdtest.NewTestApiClient(t, nil, "", glinstance.DefaultHostname).Lab()),
	)
	cmd := NewCmdCreateStack(factory, mockCmd)
	return cmdtest.ExecuteCommand(cmd, args, stdout, stderr)
}

func TestCreateNewStack(t *testing.T) {
	// NOTE: we need to force disable colors, otherwise we'd need ANSI sequences in our test output assertions.
	t.Setenv("NO_COLOR", "true")

	tests := []struct {
		desc           string
		branch         string
		expectedBranch string
		baseBranch     string
		warning        bool
	}{
		{
			desc:           "basic method",
			branch:         "test description here",
			baseBranch:     "main",
			expectedBranch: "test-description-here",
			warning:        false,
		},
		{
			desc:           "empty string",
			branch:         "",
			baseBranch:     "master",
			expectedBranch: "oh-ok-fine-how-about-blah-blah",
			warning:        true,
		},
		{
			desc:           "weird characters git won't like",
			branch:         "hey@#$!^$#)()*1234hmm",
			baseBranch:     "hello",
			expectedBranch: "hey-1234hmm",
			warning:        true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.desc, func(t *testing.T) {
			tempDir := git.InitGitRepo(t)

			ctrl := gomock.NewController(t)
			mockCmd := git_testing.NewMockGitRunner(ctrl)
			mockCmd.EXPECT().Git([]string{"symbolic-ref", "--quiet", "--short", "HEAD"}).Return(tc.baseBranch, nil)

			if tc.branch == "" {
				as, restoreAsk := prompt.InitAskStubber()
				defer restoreAsk()

				as.Stub([]*prompt.QuestionStub{
					{
						Name:  "title",
						Value: "oh ok fine how about blah blah",
					},
				})
			}

			output, err := runCommand(t, mockCmd, tc.branch)
			require.Nil(t, err)

			require.Equal(t, "New stack created with title \""+tc.expectedBranch+"\".\n", output.String())

			if tc.warning == true {
				require.Equal(t, "! warning: invalid characters have been replaced with dashes: "+tc.expectedBranch+"\n", output.Stderr())
			} else {
				require.Empty(t, output.Stderr())
			}

			configValue, err := git.GetCurrentStackTitle()
			require.Nil(t, err)

			createdBaseFile := path.Join(
				tempDir,
				"/.git/stacked/",
				tc.expectedBranch,
				git.BaseBranchFile,
			)

			fileContents, err := config.TrimmedFileContents(createdBaseFile)
			require.NoError(t, err)

			require.Equal(t, tc.baseBranch, fileContents)
			require.Equal(t, tc.expectedBranch, configValue)
			require.FileExists(t, createdBaseFile)
		})
	}
}
