//go:build !integration

package create

import (
	"path"
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/require"
	"github.com/survivorbat/huhtest"
	"go.uber.org/mock/gomock"

	"gitlab.com/gitlab-org/cli/internal/cmdutils"
	"gitlab.com/gitlab-org/cli/internal/config"
	"gitlab.com/gitlab-org/cli/internal/git"
	git_testing "gitlab.com/gitlab-org/cli/internal/git/testing"
	"gitlab.com/gitlab-org/cli/internal/glinstance"
	"gitlab.com/gitlab-org/cli/internal/testing/cmdtest"
	"gitlab.com/gitlab-org/cli/test"
)

func runCommand(t *testing.T, mockCmd git.GitRunner, args string, responder ...*huhtest.Responder) (*test.CmdOut, error) {
	t.Helper()

	// When using responder, we need to use SetupCmdForTest pattern
	if len(responder) > 0 && responder[0] != nil {
		exec := cmdtest.SetupCmdForTest(t, func(f cmdutils.Factory) *cobra.Command {
			return NewCmdCreateStack(f, mockCmd)
		}, true,
			cmdtest.WithGitLabClient(cmdtest.NewTestApiClient(t, nil, "", glinstance.DefaultHostname).Lab()),
			cmdtest.WithResponder(t, responder[0]),
		)
		return exec(args)
	}

	// Original path for non-responder tests
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
		needsResponder bool
		responderInput string
	}{
		{
			desc:           "basic method",
			branch:         "test description here",
			baseBranch:     "main",
			expectedBranch: "test-description-here",
			warning:        false,
			needsResponder: false,
		},
		{
			desc:           "empty string",
			branch:         "",
			baseBranch:     "master",
			expectedBranch: "oh-ok-fine-how-about-blah-blah",
			warning:        true,
			needsResponder: true,
			responderInput: "oh ok fine how about blah blah",
		},
		{
			desc:           "weird characters git won't like",
			branch:         "hey@#$!^$#)()*1234hmm",
			baseBranch:     "hello",
			expectedBranch: "hey-1234hmm",
			warning:        true,
			needsResponder: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.desc, func(t *testing.T) {
			tempDir := git.InitGitRepo(t)

			ctrl := gomock.NewController(t)
			mockCmd := git_testing.NewMockGitRunner(ctrl)
			mockCmd.EXPECT().Git([]string{"symbolic-ref", "--quiet", "--short", "HEAD"}).Return(tc.baseBranch, nil)

			var output *test.CmdOut
			var err error

			if tc.needsResponder {
				responder := huhtest.NewResponder()
				responder.AddResponse("New stack title?", tc.responderInput)
				output, err = runCommand(t, mockCmd, tc.branch, responder)
			} else {
				output, err = runCommand(t, mockCmd, tc.branch)
			}

			require.Nil(t, err)

			// When using responder, output may contain ANSI codes from huh prompt, so use Contains
			if tc.needsResponder {
				require.Contains(t, output.String(), "New stack created with title \""+tc.expectedBranch+"\".\n")
			} else {
				require.Equal(t, "New stack created with title \""+tc.expectedBranch+"\".\n", output.String())
			}

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
