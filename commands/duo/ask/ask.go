package ask

import (
	"errors"
	"fmt"
	"net/http"
	"os/exec"
	"regexp"
	"strings"

	"gitlab.com/gitlab-org/cli/commands/cmdutils"
	"gitlab.com/gitlab-org/cli/internal/run"
	"gitlab.com/gitlab-org/cli/pkg/iostreams"
	"gitlab.com/gitlab-org/cli/pkg/prompt"

	"github.com/MakeNowJust/heredoc/v2"
	"github.com/google/shlex"
	"github.com/spf13/cobra"
	"github.com/xanzy/go-gitlab"
)

type request struct {
	Prompt string `json:"prompt"`
	Model  string `json:"model"`
}

type response struct {
	Predictions []struct {
		Candidates []struct {
			Content string `json:"content"`
		} `json:"candidates"`
	} `json:"predictions"`
}

type result struct {
	Commands    []string `json:"commands"`
	Explanation string   `json:"explanation"`
}

type opts struct {
	Prompt     string
	IO         *iostreams.IOStreams
	HttpClient func() (*gitlab.Client, error)
	Git        bool
}

var (
	cmdHighlightRegexp = regexp.MustCompile("`+\n?([^`]*)\n?`+\n?")
	cmdExecRegexp      = regexp.MustCompile("```([^`]*)```")
	vertexAI           = "vertexai"
)

const (
	runCmdsQuestion   = "Would you like to run these Git commands?"
	gitCmd            = "git"
	gitCmdAPIPath     = "ai/llm/git_command"
	spinnerText       = "Generating Git commands..."
	aiResponseErr     = "Error: AI response has not been generated correctly."
	apiUnreachableErr = "Error: API is unreachable."
)

func NewCmdAsk(f *cmdutils.Factory) *cobra.Command {
	opts := &opts{
		IO:         f.IO,
		HttpClient: f.HttpClient,
	}

	duoAskCmd := &cobra.Command{
		Use:   "ask <prompt>",
		Short: "Generate Git commands from natural language.",
		Long: heredoc.Doc(`
			Generate Git commands from natural language.
		`),
		Example: heredoc.Doc(`
			$ glab duo ask list last 10 commit titles

			# => A list of Git commands to show the titles of the latest 10 commits with an explanation and an option to execute the commands.
		`),
		RunE: func(cmd *cobra.Command, args []string) error {
			if !opts.Git {
				return nil
			}

			if len(args) == 0 {
				return nil
			}

			opts.Prompt = strings.Join(args, " ")

			result, err := opts.Result()
			if err != nil {
				return err
			}

			opts.displayResult(result)

			if len(result.Commands) > 0 {
				if err := opts.executeCommands(result.Commands); err != nil {
					return err
				}
			}
			return nil
		},
	}

	duoAskCmd.Flags().BoolVarP(&opts.Git, "git", "", true, "Ask a question about Git.")

	return duoAskCmd
}

func (opts *opts) Result() (*result, error) {
	opts.IO.StartSpinner(spinnerText)
	defer opts.IO.StopSpinner("")

	client, err := opts.HttpClient()
	if err != nil {
		return nil, cmdutils.WrapError(err, "failed to get HTTP client.")
	}

	body := request{Prompt: opts.Prompt, Model: vertexAI}
	request, err := client.NewRequest(http.MethodPost, gitCmdAPIPath, body, nil)
	if err != nil {
		return nil, cmdutils.WrapError(err, "failed to create a request.")
	}

	var r response
	_, err = client.Do(request, &r)
	if err != nil {
		return nil, cmdutils.WrapError(err, apiUnreachableErr)
	}

	if len(r.Predictions) == 0 || len(r.Predictions[0].Candidates) == 0 {
		return nil, errors.New(aiResponseErr)
	}

	content := r.Predictions[0].Candidates[0].Content

	var cmds []string
	for _, cmd := range cmdExecRegexp.FindAllString(content, -1) {
		cmds = append(cmds, strings.Trim(cmd, "\n`"))
	}

	return &result{
		Commands:    cmds,
		Explanation: content,
	}, nil
}

func (opts *opts) displayResult(result *result) {
	color := opts.IO.Color()

	opts.IO.LogInfo(color.Bold("Commands:\n"))

	for _, cmd := range result.Commands {
		opts.IO.LogInfo(color.Green(cmd))
	}

	opts.IO.LogInfo(color.Bold("\nExplanation:\n"))
	explanation := cmdHighlightRegexp.ReplaceAllString(result.Explanation, color.Green("$1"))
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
