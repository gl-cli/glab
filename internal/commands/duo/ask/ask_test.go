package ask

import (
	"net/http"
	"testing"

	"gitlab.com/gitlab-org/cli/internal/glinstance"
	"gitlab.com/gitlab-org/cli/internal/prompt"

	"gitlab.com/gitlab-org/cli/internal/testing/cmdtest"

	"github.com/stretchr/testify/require"
	"gitlab.com/gitlab-org/cli/internal/testing/httpmock"
	"gitlab.com/gitlab-org/cli/test"
)

func runCommand(t *testing.T, rt http.RoundTripper, args string, glInstanceHostname string) (*test.CmdOut, *cmdtest.Factory, error) {
	ios, _, stdout, stderr := cmdtest.TestIOStreams()

	factory := cmdtest.NewTestFactory(ios,
		cmdtest.WithApiClient(cmdtest.NewTestApiClient(t, &http.Client{Transport: rt}, "", glInstanceHostname)),
		cmdtest.WithBaseRepo("OWNER", "REPO", glInstanceHostname),
	)

	cmd := NewCmdAsk(factory)

	cmdOut, err := cmdtest.ExecuteCommand(cmd, args, stdout, stderr)

	return cmdOut, factory, err
}

func TestAskCmd(t *testing.T) {
	initialAiResponse := "The appropriate ```git log --pretty=format:'%h'``` Git command ```non-git cmd``` for listing ```git show``` commit SHAs."
	outputWithoutExecution := "Commands:\n" + `
git log --pretty=format:'%h'
non-git cmd
git show

Explanation:

The appropriate git log --pretty=format:'%h' Git command non-git cmd for listing git show commit SHAs.

`

	tests := []struct {
		desc                       string
		content                    string
		withPrompt                 bool
		withExecution              bool
		withGlInstanceHostname     string
		expectedResult             string
		expectedGlInstanceHostname string
	}{
		{
			desc:                       "agree to run commands",
			content:                    initialAiResponse,
			withGlInstanceHostname:     "",
			withPrompt:                 true,
			withExecution:              true,
			expectedResult:             outputWithoutExecution + "git log executed\ngit show executed\n",
			expectedGlInstanceHostname: glinstance.DefaultHostname,
		},
		{
			desc:                       "disagree to run commands",
			content:                    initialAiResponse,
			withGlInstanceHostname:     "example.com",
			withPrompt:                 true,
			withExecution:              false,
			expectedResult:             outputWithoutExecution,
			expectedGlInstanceHostname: "example.com",
		},
		{
			desc:                       "no commands",
			content:                    "There are no Git commands related to the text.",
			withGlInstanceHostname:     "instance.example.com",
			withPrompt:                 false,
			expectedResult:             "Commands:\n\n\nExplanation:\n\nThere are no Git commands related to the text.\n\n",
			expectedGlInstanceHostname: "instance.example.com",
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

			body := `{"predictions": [{ "candidates": [ {"content": "` + tc.content + `"} ]}]}`

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

			output, factory, err := runCommand(t, fakeHTTP, "git list 10 commits", tc.withGlInstanceHostname)
			baseRepo, _ := factory.BaseRepo()
			require.Nil(t, err)

			require.Equal(t, output.String(), tc.expectedResult)
			require.Empty(t, output.Stderr())
			require.Equal(t, baseRepo.RepoHost(), tc.expectedGlInstanceHostname)
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
			expectedMsg: "404 Not Found",
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

			_, _, err := runCommand(t, fakeHTTP, "git list 10 commits", "")
			require.NotNil(t, err)
			require.Contains(t, err.Error(), tc.expectedMsg)
		})
	}
}
