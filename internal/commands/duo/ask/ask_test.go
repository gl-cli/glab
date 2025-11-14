//go:build !integration

package ask

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/survivorbat/huhtest"

	"gitlab.com/gitlab-org/cli/internal/api"
	"gitlab.com/gitlab-org/cli/internal/glinstance"
	"gitlab.com/gitlab-org/cli/internal/testing/cmdtest"
	"gitlab.com/gitlab-org/cli/internal/testing/httpmock"
	"gitlab.com/gitlab-org/cli/test"
)

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
		expectedResult             []string
		expectedGlInstanceHostname string
	}{
		{
			desc:                       "agree to run commands",
			content:                    initialAiResponse,
			withGlInstanceHostname:     "",
			withPrompt:                 true,
			withExecution:              true,
			expectedResult:             []string{outputWithoutExecution, "git log executed", "git show executed"},
			expectedGlInstanceHostname: glinstance.DefaultHostname,
		},
		{
			desc:                       "disagree to run commands",
			content:                    initialAiResponse,
			withGlInstanceHostname:     "example.com",
			withPrompt:                 true,
			withExecution:              false,
			expectedResult:             []string{outputWithoutExecution},
			expectedGlInstanceHostname: "example.com",
		},
		{
			desc:                       "no commands",
			content:                    "There are no Git commands related to the text.",
			withGlInstanceHostname:     "instance.example.com",
			withPrompt:                 false,
			withExecution:              false,
			expectedResult:             []string{"Commands:\n\n\nExplanation:\n\nThere are no Git commands related to the text.\n\n"},
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
				cs, restore := test.InitCmdStubber()
				defer restore()
				cs.Stub(cmdLogResult)
				cs.Stub(cmdShowResult)
			}

			opts := []cmdtest.FactoryOption{
				func(f *cmdtest.Factory) {
					f.ApiClientStub = func(repoHost string) (*api.Client, error) {
						require.Equal(t, tc.expectedGlInstanceHostname, repoHost)

						return cmdtest.NewTestApiClient(t, &http.Client{Transport: fakeHTTP}, "", tc.withGlInstanceHostname), nil
					}
				},
				cmdtest.WithBaseRepo("OWNER", "REPO", tc.withGlInstanceHostname),
			}

			// Set up prompt stub if needed
			if tc.withPrompt {
				responder := huhtest.NewResponder()
				// FIXME: there is a bug in huhtest (I've created https://github.com/survivorbat/huhtest/issues/2)
				// which leads to wrong answers when the Confirm has an affirmative default.
				// Therefore, we need to invert our actual answer.
				if !tc.withExecution {
					responder = responder.
						AddConfirm(runCmdsQuestion, huhtest.ConfirmAffirm).
						AddConfirm("Run `.*?`", huhtest.ConfirmAffirm).MatchRegexp()
				} else {
					responder = responder.
						AddConfirm(runCmdsQuestion, huhtest.ConfirmNegative).
						AddConfirm("Run `.*?`", huhtest.ConfirmNegative).MatchRegexp()
				}
				opts = append(opts, cmdtest.WithResponder(t, responder))
			}

			exec := cmdtest.SetupCmdForTest(t, NewCmdAsk, false, opts...)

			output, err := exec("git list 10 commits")
			require.NoError(t, err)

			stdout := output.String()
			for _, r := range tc.expectedResult {
				assert.Contains(t, stdout, r)
			}
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

			exec := cmdtest.SetupCmdForTest(t, NewCmdAsk, false,
				cmdtest.WithApiClient(cmdtest.NewTestApiClient(t, &http.Client{Transport: fakeHTTP}, "", "")),
				cmdtest.WithBaseRepo("OWNER", "REPO", ""),
			)

			_, err := exec("git list 10 commits")
			require.EqualError(t, err, tc.expectedMsg)
		})
	}
}
