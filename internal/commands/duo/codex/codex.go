package codex

import (
	"fmt"
	"os"
	"os/exec"

	"gitlab.com/gitlab-org/cli/internal/api"
	"gitlab.com/gitlab-org/cli/internal/cmdutils"
	"gitlab.com/gitlab-org/cli/internal/commands/duo/utils"
	"gitlab.com/gitlab-org/cli/internal/glrepo"
	"gitlab.com/gitlab-org/cli/internal/iostreams"
	"gitlab.com/gitlab-org/cli/internal/text"
	"gitlab.com/gitlab-org/cli/internal/thirdpartyagents"

	"github.com/MakeNowJust/heredoc/v2"
	"github.com/spf13/cobra"
)

const CodexExecutableName = "codex"

// opts holds the configuration options for Codex commands.
type opts struct {
	Prompt    string
	IO        *iostreams.IOStreams
	apiClient func(repoHost string) (*api.Client, error)
	BaseRepo  func() (glrepo.Interface, error)
}

func NewCmdCodex(f cmdutils.Factory) *cobra.Command {
	opts := &opts{
		IO:        f.IO(),
		apiClient: f.ApiClient,
		BaseRepo:  f.BaseRepo,
	}

	duoCodexCmd := &cobra.Command{
		Use:   "codex [flags] [args]",
		Short: "Launch Codex with GitLab Duo integration (EXPERIMENTAL)",
		Long: heredoc.Doc(`
			Launch Codex with automatic GitLab authentication and proxy configuration.
			All flags and arguments are passed through to the Codex executable.
			
			This command automatically configures Codex to work with GitLab AI services,
			handling authentication tokens and API endpoints based on your current repository.
		`) + text.ExperimentalString,
		Example: heredoc.Doc(`
			$ glab duo codex
		`),
		// Allow unknown flags to be passed through to the Codex command
		FParseErrWhitelist: cobra.FParseErrWhitelist{
			UnknownFlags: true,
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			// Fetch repo host
			var repoHost string
			if baseRepo, err := opts.BaseRepo(); err == nil {
				repoHost = baseRepo.RepoHost()
			}

			// Get API client
			c, err := opts.apiClient(repoHost)
			if err != nil {
				return err
			}

			// Fetch direct_access token
			token, err := thirdpartyagents.FetchDirectAccessToken(c.Lab())
			if err != nil {
				return fmt.Errorf("failed to retrieve GitLab Duo access token: %w", err)
			}

			// Validate Codex executable exists
			if err := utils.ValidateExecutable(CodexExecutableName); err != nil {
				return fmt.Errorf("codex executable validation failed: %w", err)
			}

			// Create codex config
			configArgs := createCodexConfigArgs(token.Headers)

			// Extract codex command arguments
			codexArgs, err := utils.ExtractArgs(CodexExecutableName)
			if err != nil {
				return fmt.Errorf("failed to parse command arguments: %w", err)
			}

			// Add --config flag to codex arguments
			codexArgs = append(configArgs, codexArgs...)

			// Execute codex command with all arguments
			codexCmd := exec.Command(CodexExecutableName, codexArgs...)

			// Connect standard input/output/error
			codexCmd.Stdin = opts.IO.In
			codexCmd.Stdout = opts.IO.StdOut
			codexCmd.Stderr = opts.IO.StdErr

			// Set environment variables for the codex command
			codexCmd.Env = append(os.Environ(), fmt.Sprintf("%s=%s", "OPENAI_API_KEY", token.Token))

			// Execute the command
			if err := codexCmd.Run(); err != nil {
				return fmt.Errorf("failed to execute codex: %w", err)
			}

			return nil
		},
	}

	return duoCodexCmd
}
