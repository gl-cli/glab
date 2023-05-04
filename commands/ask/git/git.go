package git

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os/exec"
	"regexp"
	"strings"

	"gitlab.com/gitlab-org/cli/commands/cmdutils"
	"gitlab.com/gitlab-org/cli/internal/run"
	"gitlab.com/gitlab-org/cli/pkg/iostreams"
	"gitlab.com/gitlab-org/cli/pkg/prompt"

	"github.com/MakeNowJust/heredoc"
	"github.com/google/shlex"
	"github.com/spf13/cobra"
	"github.com/xanzy/go-gitlab"
)

type request struct {
	Prompt string `json:"prompt"`
}

type response struct {
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
}

type result struct {
	Commands    []string `json:"commands"`
	Explanation string   `json:"explanation"`
}

type opts struct {
	Prompt     string
	IO         *iostreams.IOStreams
	HttpClient func() (*gitlab.Client, error)
}

var cmdRegexp = regexp.MustCompile("`([^`]*)`")

const (
	runCmdsQuestion   = "Would you like to run these Git commands"
	gitCmd            = "git"
	gitCmdAPIPath     = "ai/llm/git_command"
	spinnerText       = "Generating Git commands..."
	aiResponseErr     = "Error: AI response has not been generated correctly"
	apiUnreachableErr = "Error: API is unreachable"
)

func NewCmd(f *cmdutils.Factory) *cobra.Command {
	opts := &opts{
		IO:         f.IO,
		HttpClient: f.HttpClient,
	}

	cmd := &cobra.Command{
		Use:   "git <prompt>",
		Short: "Generate Git commands from natural language (Experimental).",
		Long: heredoc.Doc(`
			Generate Git commands from natural language.

			This experimental feature converts natural language descriptions into
			executable Git commands.

			We'd love your feedback in [issue 409636](https://gitlab.com/gitlab-org/gitlab/-/issues/409636).
		`),
		Example: heredoc.Doc(`
			$ glab ask git list last 10 commit titles
			# => A list of Git commands to show the titles of the latest 10 commits with an explanation and an option to execute the commands.
		`),
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 {
				return nil
			}

			opts.Prompt = strings.Join(args, " ")

			result, err := opts.Result()
			if err != nil {
				return err
			}

			opts.displayResult(result)

			if err := opts.executeCommands(result.Commands); err != nil {
				return err
			}

			return nil
		},
	}

	return cmd
}

func (opts *opts) Result() (*result, error) {
	opts.IO.StartSpinner(spinnerText)
	defer opts.IO.StopSpinner("")

	client, err := opts.HttpClient()
	if err != nil {
		return nil, cmdutils.WrapError(err, "failed to get http client")
	}

	body := request{Prompt: opts.Prompt}
	request, err := client.NewRequest(http.MethodPost, gitCmdAPIPath, body, nil)
	if err != nil {
		return nil, cmdutils.WrapError(err, "failed to create a request")
	}

	var r response
	_, err = client.Do(request, &r)
	if err != nil {
		return nil, cmdutils.WrapError(err, apiUnreachableErr)
	}

	if len(r.Choices) == 0 {
		return nil, fmt.Errorf(aiResponseErr)
	}

	var result result
	if err := json.Unmarshal([]byte(r.Choices[0].Message.Content), &result); err != nil {
		return nil, fmt.Errorf(aiResponseErr)
	}

	return &result, nil
}

func (opts *opts) displayResult(result *result) {
	color := opts.IO.Color()

	opts.IO.LogInfo(color.Bold("Commands:\n"))

	for _, cmd := range result.Commands {
		opts.IO.LogInfo(color.Green(cmd))
	}

	opts.IO.LogInfo(color.Bold("\nExplanation:\n"))
	explanation := cmdRegexp.ReplaceAllString(result.Explanation, color.Green("$1"))
	opts.IO.LogInfo(explanation + "\n")
}

func (opts *opts) executeCommands(commands []string) error {
	color := opts.IO.Color()

	var confirmed bool
	question := color.Bold(runCmdsQuestion)
	if err := prompt.Confirm(&confirmed, question, true); err != nil {
		return err
	}

	if !confirmed {
		return nil
	}

	for _, command := range commands {
		if err := opts.executeCommand(command); err != nil {
			return err
		}
	}

	return nil
}

func (opts *opts) executeCommand(cmd string) error {
	gitArgs, err := shlex.Split(cmd)
	if err != nil {
		return nil
	}

	if gitArgs[0] != gitCmd {
		return nil
	}

	color := opts.IO.Color()
	question := fmt.Sprintf("Run `%s`", color.Green(cmd))
	var confirmed bool
	if err := prompt.Confirm(&confirmed, question, true); err != nil {
		return err
	}

	if !confirmed {
		return nil
	}

	execCmd := exec.Command("git", gitArgs[1:]...)
	output, err := run.PrepareCmd(execCmd).Output()
	if err != nil {
		return err
	}

	if len(output) == 0 {
		return nil
	}

	if opts.IO.StartPager() != nil {
		return fmt.Errorf("failed to start pager: %q", err)
	}
	defer opts.IO.StopPager()

	opts.IO.LogInfo(string(output))

	return nil
}
