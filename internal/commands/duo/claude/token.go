package claude

import (
	"fmt"

	"gitlab.com/gitlab-org/cli/internal/mcpannotations"

	"gitlab.com/gitlab-org/cli/internal/cmdutils"

	"github.com/MakeNowJust/heredoc/v2"
	"github.com/spf13/cobra"
)

// NewCmdToken creates a new cobra command for generating GitLab Duo access tokens.
func NewCmdToken(f cmdutils.Factory) *cobra.Command {
	opts := &opts{
		IO:        f.IO(),
		apiClient: f.ApiClient,
		BaseRepo:  f.BaseRepo,
	}

	duoClaudeTokenCmd := &cobra.Command{
		Use:   "token",
		Short: "Generate GitLab Duo access token for Claude Code",
		Long: heredoc.Doc(`
			Generate and display a GitLab Duo access token required for Claude Code authentication.
			
			This token allows Claude Code to authenticate with GitLab AI services.
			The token is automatically used when running 'glab duo claude'.
		`),
		Example: heredoc.Doc(`
			$ glab duo claude token
		`),
		// Allow unknown flags to be passed through to the claude command
		FParseErrWhitelist: cobra.FParseErrWhitelist{
			UnknownFlags: true,
		},
		Annotations: map[string]string{
			mcpannotations.Destructive: "true",
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
			token, err := fetchDirectAccessToken(c.Lab())
			if err != nil {
				return fmt.Errorf("failed to retrieve GitLab Duo access token: %w", err)
			}

			fmt.Fprintln(opts.IO.StdOut, token.Token)

			return nil
		},
	}

	return duoClaudeTokenCmd
}
