package list

import (
	"encoding/json"
	"errors"
	"strconv"
	"strings"
	"time"

	"github.com/MakeNowJust/heredoc/v2"
	"github.com/spf13/cobra"
	gitlab "gitlab.com/gitlab-org/api/client-go"

	"gitlab.com/gitlab-org/cli/api"
	"gitlab.com/gitlab-org/cli/commands/cmdutils"
	"gitlab.com/gitlab-org/cli/commands/flag"
	"gitlab.com/gitlab-org/cli/commands/token/accesslevel"

	"gitlab.com/gitlab-org/cli/internal/glrepo"
	"gitlab.com/gitlab-org/cli/pkg/iostreams"
)

type ListOpts struct {
	HTTPClient func() (*gitlab.Client, error)
	IO         *iostreams.IOStreams
	BaseRepo   func() (glrepo.Interface, error)

	User         string
	Group        string
	OutputFormat string
	ListActive   bool
}

func NewCmdList(f cmdutils.Factory, runE func(opts *ListOpts) error) *cobra.Command {
	opts := &ListOpts{
		IO: f.IO(),
	}

	cmd := &cobra.Command{
		Use:     "list",
		Short:   "List user, group, or project access tokens.",
		Aliases: []string{"ls"},
		Args:    cobra.ExactArgs(0),
		Long: heredoc.Doc(`
			List all tokens of a user, group, or project.

			The output contains the token's meta information, not the actual token value. The output format
			can be "JSON" or "text". The access level property is printed in human-readable form in the text
			output, but displays the integer value in JSON.

			Administrators can list tokens of other users.
		`),
		Example: heredoc.Doc(
			`
			List the current project's access tokens
			- glab token list
			- glab token list --output json

			List the project access tokens of a specific project
			- glab token list --repo user/my-repo

			List group access tokens
			- glab token list --group group/sub-group

			List my personal access tokens
			- glab token list --user @me

			Administrators only: list the personal access tokens of another user
			- glab token list --user johndoe
		`,
		),
		RunE: func(cmd *cobra.Command, args []string) (err error) {
			opts.HTTPClient = f.HttpClient
			opts.BaseRepo = f.BaseRepo

			opts.Group, err = flag.GroupOverride(cmd)
			if err != nil {
				return err
			}

			if opts.Group != "" && opts.User != "" {
				return cmdutils.FlagError{Err: errors.New("--user and --group are mutually exclusive.")}
			}

			if runE != nil {
				return runE(opts)
			}
			return listRun(opts)
		},
	}

	cmdutils.EnableRepoOverride(cmd, f)
	cmd.Flags().StringVarP(&opts.Group, "group", "g", "", "List group access tokens. Ignored if a user or repository argument is set.")
	cmd.Flags().StringVarP(&opts.User, "user", "U", "", "List personal access tokens. Use @me for the current user.")
	cmd.Flags().StringVarP(&opts.OutputFormat, "output", "F", "text", "Format output as: text, json. text provides a readable table, json outputs the tokens with metadata.")
	cmd.Flags().BoolVarP(&opts.ListActive, "active", "a", false, "List only the active tokens.")

	return cmd
}

type Token struct {
	ID          string
	Name        string
	Description string
	AccessLevel string
	Active      string
	Revoked     string
	CreatedAt   string
	ExpiresAt   string
	LastUsedAt  string
	Scopes      string
}

type Tokens []Token

func formatLastUsedAt(lastUsedAt *time.Time) string {
	if lastUsedAt == nil {
		return "-"
	}
	return lastUsedAt.Format(time.RFC3339)
}

func formatDescription(description string) string {
	if description == "" {
		return "-"
	}
	return description
}

func formatAccessLevel(accessLevel gitlab.AccessLevelValue) string {
	level := accesslevel.AccessLevel{Value: accessLevel}
	levelStr := level.String()

	if levelStr == "no" {
		return "-"
	}

	return levelStr
}

func formatExpiresAt(expiresAt *gitlab.ISOTime) string {
	if expiresAt == nil {
		return "-"
	}
	return expiresAt.String()
}

func listRun(opts *ListOpts) error {
	httpClient, err := opts.HTTPClient()
	if err != nil {
		return err
	}

	var apiTokens any
	var outputTokens Tokens
	switch {
	case opts.User != "":
		user, err := api.UserByName(httpClient, opts.User)
		if err != nil {
			return err
		}
		options := &gitlab.ListPersonalAccessTokensOptions{
			UserID: &user.ID,
		}
		tokens, err := api.ListPersonalAccessTokens(httpClient, options)
		if err != nil {
			return err
		}
		apiTokens = tokens
		outputTokens = make([]Token, 0, len(tokens))
		for _, token := range tokens {
			if !opts.ListActive || token.Active {
				outputTokens = append(outputTokens, Token{
					ID:          strconv.FormatInt(int64(token.ID), 10),
					Name:        token.Name,
					Description: formatDescription(token.Description),
					AccessLevel: "-",
					Active:      strconv.FormatBool(token.Active),
					Revoked:     strconv.FormatBool(token.Revoked),
					CreatedAt:   token.CreatedAt.Format(time.RFC3339),
					ExpiresAt:   formatExpiresAt(token.ExpiresAt),
					LastUsedAt:  formatLastUsedAt(token.LastUsedAt),
					Scopes:      strings.Join(token.Scopes, ","),
				})
			}
		}
	case opts.Group != "":
		options := &gitlab.ListGroupAccessTokensOptions{}
		tokens, err := api.ListGroupAccessTokens(httpClient, opts.Group, options)
		if err != nil {
			return err
		}
		apiTokens = tokens
		outputTokens = make([]Token, 0, len(tokens))
		for _, token := range tokens {
			if !opts.ListActive || token.Active {
				outputTokens = append(outputTokens, Token{
					ID:          strconv.FormatInt(int64(token.ID), 10),
					Name:        token.Name,
					Description: formatDescription(token.Description),
					AccessLevel: formatAccessLevel(token.AccessLevel),
					Active:      strconv.FormatBool(token.Active),
					Revoked:     strconv.FormatBool(token.Revoked),
					CreatedAt:   token.CreatedAt.Format(time.RFC3339),
					ExpiresAt:   formatExpiresAt(token.ExpiresAt),
					LastUsedAt:  formatLastUsedAt(token.LastUsedAt),
					Scopes:      strings.Join(token.Scopes, ","),
				})
			}
		}
	default:
		repo, err := opts.BaseRepo()
		if err != nil {
			return err
		}

		tokens, err := api.ListProjectAccessTokens(httpClient, repo.FullName(), &gitlab.ListProjectAccessTokensOptions{})
		if err != nil {
			return err
		}
		apiTokens = tokens
		outputTokens = make([]Token, 0, len(tokens))
		for _, token := range tokens {
			if !opts.ListActive || token.Active {
				outputTokens = append(outputTokens, Token{
					ID:          strconv.FormatInt(int64(token.ID), 10),
					Name:        token.Name,
					Description: formatDescription(token.Description),
					AccessLevel: formatAccessLevel(token.AccessLevel),
					Active:      strconv.FormatBool(token.Active),
					Revoked:     strconv.FormatBool(token.Revoked),
					CreatedAt:   token.CreatedAt.Format(time.RFC3339),
					ExpiresAt:   formatExpiresAt(token.ExpiresAt),
					LastUsedAt:  formatLastUsedAt(token.LastUsedAt),
					Scopes:      strings.Join(token.Scopes, ","),
				})
			}
		}
	}

	if opts.OutputFormat == "json" {
		encoder := json.NewEncoder(opts.IO.StdOut)
		if err := encoder.Encode(apiTokens); err != nil {
			return err
		}
	} else {
		table := createTablePrinter(outputTokens)
		opts.IO.LogInfof("%s", table.String())
	}
	return nil
}
