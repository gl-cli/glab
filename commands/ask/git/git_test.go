package git

import (
	"net/http"
	"testing"

	"gitlab.com/gitlab-org/cli/pkg/prompt"

	"gitlab.com/gitlab-org/cli/commands/cmdtest"

	"github.com/stretchr/testify/require"
	"gitlab.com/gitlab-org/cli/pkg/httpmock"
	"gitlab.com/gitlab-org/cli/test"
)

func runCommand(rt http.RoundTripper, isTTY bool, args string) (*test.CmdOut, error) {
	ios, _, stdout, stderr := cmdtest.InitIOStreams(isTTY, "")

	factory := cmdtest.InitFactory(ios, rt)

	_, _ = factory.HttpClient()

	cmd := NewCmd(factory)

	return cmdtest.ExecuteCommand(cmd, args, stdout, stderr)
}

func TestGitCmd(t *testing.T) {
	outputWithoutExecution := `Commands:

git log --pretty=format:'%h'
non-git cmd
git show

Explanation:

The appropriate Git command for listing commit SHAs.

`
	tests := []struct {
		desc           string
		content        string
		withPrompt     bool
		withExecution  bool
		expectedResult string
	}{
		{
			desc:           "agree to run commands",
			content:        `{\"commands\": [\"git log --pretty=format:'%h'\", \"non-git cmd\", \"git show\"], \"explanation\":\"The appropriate Git command for listing commit SHAs.\"}`,
			withPrompt:     true,
			withExecution:  true,
			expectedResult: outputWithoutExecution + "git log executed\ngit show executed\n",
		},
		{
			desc:           "disagree to run commands",
			content:        `{\"commands\": [\"git log --pretty=format:'%h'\", \"non-git cmd\", \"git show\"], \"explanation\":\"The appropriate Git command for listing commit SHAs.\"}`,
			withPrompt:     true,
			withExecution:  false,
			expectedResult: outputWithoutExecution,
		},
		{
			desc:           "no commands",
			content:        `{\"commands\": [], \"explanation\":\"There are no git commands related to the text\"}`,
			withPrompt:     false,
			expectedResult: "Commands:\n\n\nExplanation:\n\nThere are no git commands related to the text\n\n",
		},
	}
	cmdLogResult := "git log executed"
	cmdShowResult := "git show executed"

	for _, tc := range tests {
		t.Run(tc.desc, func(t *testing.T) {
			fakeHTTP := &httpmock.Mocker{
				MatchURL: httpmock.PathAndQuerystring,
			}
			defer fakeHTTP.Verify(t)

			body := `{"choices": [{"message": {"content": "` + tc.content + `"}}]}`

			response := httpmock.NewStringResponse(http.StatusOK, body)
			fakeHTTP.RegisterResponder(http.MethodPost, "/api/v4/ai/llm/git_command", response)

			if tc.withPrompt {
				restore := prompt.StubConfirm(tc.withExecution)
				defer restore()

				cs, restore := test.InitCmdStubber()
				defer restore()
				cs.Stub(cmdLogResult)
				cs.Stub(cmdShowResult)
			}

			output, err := runCommand(fakeHTTP, false, "git list 10 commits")
			require.Nil(t, err)

			require.Equal(t, tc.expectedResult, output.String())
			require.Empty(t, output.Stderr())
		})
	}
}

func TestFailedHttpResponse(t *testing.T) {
	tests := []struct {
		desc        string
		code        int
		response    string
		expectedMsg string
	}{
		{
			desc:        "API error",
			code:        http.StatusNotFound,
			response:    `{"message": "Error message"}`,
			expectedMsg: "404 {message: Error message}",
		},
		{
			desc:        "Empty response",
			code:        http.StatusOK,
			response:    `{"choices": []}`,
			expectedMsg: aiResponseErr,
		},
		{
			desc:        "Bad JSON",
			code:        http.StatusOK,
			response:    `{"choices": [{"message": {"content": "hello"}}]}`,
			expectedMsg: aiResponseErr,
		},
	}

	for _, tc := range tests {
		t.Run(tc.desc, func(t *testing.T) {
			fakeHTTP := &httpmock.Mocker{
				MatchURL: httpmock.PathAndQuerystring,
			}
			defer fakeHTTP.Verify(t)

			response := httpmock.NewStringResponse(tc.code, tc.response)
			fakeHTTP.RegisterResponder(http.MethodPost, "/api/v4/ai/llm/git_command", response)

			_, err := runCommand(fakeHTTP, false, "git list 10 commits")
			require.NotNil(t, err)
			require.Contains(t, err.Error(), tc.expectedMsg)
		})
	}
}
