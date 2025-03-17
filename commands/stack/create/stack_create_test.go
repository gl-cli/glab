package create

import (
	"net/http"
	"path"
	"testing"

	"github.com/stretchr/testify/require"
	"gitlab.com/gitlab-org/cli/commands/cmdtest"
	"gitlab.com/gitlab-org/cli/pkg/git"
	git_testing "gitlab.com/gitlab-org/cli/pkg/git/testing"
	"gitlab.com/gitlab-org/cli/pkg/prompt"
	"gitlab.com/gitlab-org/cli/test"
	"go.uber.org/mock/gomock"
)

func runCommand(rt http.RoundTripper, isTTY bool, args string, t *testing.T) (*test.CmdOut, error) {
	ios, _, stdout, stderr := cmdtest.InitIOStreams(isTTY, "")

	factory := cmdtest.InitFactory(ios, rt)

	_, _ = factory.HttpClient()

	ctrl := gomock.NewController(t)
	mockCmd := git_testing.NewMockGitRunner(ctrl)

	cmd := NewCmdCreateStack(factory, mockCmd)

	return cmdtest.ExecuteCommand(cmd, args, stdout, stderr)
}

func TestCreateNewStack(t *testing.T) {
	tests := []struct {
		desc           string
		branch         string
		expectedBranch string
		warning        bool
	}{
		{
			desc:           "basic method",
			branch:         "test description here",
			expectedBranch: "test-description-here",
			warning:        false,
		},
		{
			desc:           "empty string",
			branch:         "",
			expectedBranch: "oh-ok-fine-how-about-blah-blah",
			warning:        true,
		},
		{
			desc:           "weird characters git won't like",
			branch:         "hey@#$!^$#)()*1234hmm",
			expectedBranch: "hey-1234hmm",
			warning:        true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.desc, func(t *testing.T) {
			tempDir := git.InitGitRepo(t)

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

			output, err := runCommand(nil, true, tc.branch, t)
			require.Nil(t, err)

			require.Equal(t, "New stack created with title \""+tc.expectedBranch+"\".\n", output.String())

			if tc.warning == true {
				require.Equal(t, "! warning: invalid characters have been replaced with dashes: "+tc.expectedBranch+"\n", output.Stderr())
			} else {
				require.Empty(t, output.Stderr())
			}

			configValue, err := git.GetCurrentStackTitle()
			require.Nil(t, err)

			require.Equal(t, tc.expectedBranch, configValue)
			require.DirExists(t, path.Join(tempDir, "/.git/stacked/", tc.expectedBranch))
		})
	}
}
