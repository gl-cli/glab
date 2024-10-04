package revoke

import (
	"encoding/json"
	"errors"
	"fmt"
	"strconv"

	"gitlab.com/gitlab-org/cli/commands/token/filter"

	"gitlab.com/gitlab-org/cli/commands/flag"
	"gitlab.com/gitlab-org/cli/pkg/iostreams"

	"github.com/MakeNowJust/heredoc/v2"
	"github.com/spf13/cobra"
	"github.com/xanzy/go-gitlab"
	"gitlab.com/gitlab-org/cli/api"
	"gitlab.com/gitlab-org/cli/commands/cmdutils"
	"gitlab.com/gitlab-org/cli/internal/glrepo"
)

type RevokeOptions struct {
	HTTPClient func() (*gitlab.Client, error)
	IO         *iostreams.IOStreams
	BaseRepo   func() (glrepo.Interface, error)

	User         string
	Group        string
	Name         string
	TokenId      int
	OutputFormat string
}

func NewCmdRevoke(f *cmdutils.Factory, runE func(opts *RevokeOptions) error) *cobra.Command {
	opts := &RevokeOptions{
		IO: f.IO,
	}

	cmd := &cobra.Command{
		Use:     "revoke <token-name|token-id>",
		Short:   "Revoke user, group or project access tokens",
		Aliases: []string{"revoke", "rm"},
		Args:    cobra.RangeArgs(1, 1),
		Long: heredoc.Doc(`
			Revoke an user, group or project access token. If multiple tokens with the same name exist, you can specify
			the ID of the token.

			The output format can be either "JSON" or "text". The JSON output will show the meta information of the
			revoked token. The normal text output is a description of the revoked token name and ID.

			Administrators can revoke personal access tokens belonging to other users.
		`),

		Example: heredoc.Doc(`
			# Revoke a project access token of current project
			glab token revoke my-project-token

			# Revoke a project access token of a specific project
			glab token revoke --repo user/my-repo my-project-token

			# Revoke a group access token
			glab token revoke --group group/sub-group my-group-token

			# Revoke my personal access token
			glab token revoke --user @me my-personal-token

			# Revoke a personal access token of another user (administrator only)
			glab token revoke --user johndoe johns-personal-token

		`),
		RunE: func(cmd *cobra.Command, args []string) (err error) {
			// Supports repo override
			opts.HTTPClient = f.HttpClient
			opts.BaseRepo = f.BaseRepo

			if opts.TokenId, err = strconv.Atoi(args[0]); err != nil {
				opts.Name = args[0]
			}
			if opts.Group, err = flag.GroupOverride(cmd); err != nil {
				return
			}

			if opts.Group != "" && opts.User != "" {
				return cmdutils.FlagError{Err: errors.New("'--group' and '--user' are mutually exclusive.")}
			}

			if runE != nil {
				return runE(opts)
			}
			return revokeTokenRun(opts)
		},
	}

	cmdutils.EnableRepoOverride(cmd, f)
	cmd.Flags().StringVarP(&opts.Group, "group", "g", "", "Revoke group access token. Ignored if a user or repository argument is set.")
	cmd.Flags().StringVarP(&opts.User, "user", "U", "", "Revoke personal access token. Use @me for the current user.")
	cmd.Flags().StringVarP(&opts.OutputFormat, "output", "F", "text", "Format output as: text, json. 'text' provides the name and ID of the revoked token; 'json' outputs the token with metadata.")
	return cmd
}

func revokeTokenRun(opts *RevokeOptions) error {
	httpClient, err := opts.HTTPClient()
	if err != nil {
		return err
	}

	var outputToken interface{}
	var outputTokenValue string

	if opts.User != "" {
		user, err := api.UserByName(httpClient, opts.User)
		if err != nil {
			return cmdutils.FlagError{Err: err}
		}

		options := &gitlab.ListPersonalAccessTokensOptions{
			ListOptions: gitlab.ListOptions{PerPage: 100},
			UserID:      &user.ID,
		}
		tokens, err := api.ListPersonalAccessTokens(httpClient, options)
		if err != nil {
			return err
		}
		var token *gitlab.PersonalAccessToken
		tokens = filter.Filter(tokens, func(t *gitlab.PersonalAccessToken) bool {
			return t.Active && (t.Name == opts.Name || t.ID == opts.TokenId)
		})
		switch len(tokens) {
		case 1:
			token = tokens[0]
		case 0:
			return cmdutils.FlagError{Err: fmt.Errorf("no token found with the name '%v'.", opts.Name)}
		default:
			return cmdutils.FlagError{Err: fmt.Errorf("multiple tokens found with the name '%v'. Use the ID instead.", opts.Name)}
		}
		if err = api.RevokePersonalAccessToken(httpClient, token.ID); err != nil {
			return err
		}
		token.Revoked = true
		outputToken = token
		outputTokenValue = fmt.Sprintf("revoked %s %s %d", opts.User, token.Name, token.ID)
	} else {
		if opts.Group != "" {
			options := &gitlab.ListGroupAccessTokensOptions{PerPage: 100}
			tokens, err := api.ListGroupAccessTokens(httpClient, opts.Group, options)
			if err != nil {
				return err
			}
			var token *gitlab.GroupAccessToken
			tokens = filter.Filter(tokens, func(t *gitlab.GroupAccessToken) bool {
				return t.Active && (t.Name == opts.Name || t.ID == opts.TokenId)
			})
			switch len(tokens) {
			case 1:
				token = tokens[0]
			case 0:
				return cmdutils.FlagError{Err: fmt.Errorf("no token found with the name '%v'.", opts.Name)}
			default:
				return cmdutils.FlagError{Err: fmt.Errorf("multiple tokens found with the name '%v'. Use the ID instead.", opts.Name)}
			}

			if err = api.RevokeGroupAccessToken(httpClient, opts.Group, token.ID); err != nil {
				return err
			}
			token.Revoked = true
			outputToken = token
			outputTokenValue = fmt.Sprintf("revoked %s %d", token.Name, token.ID)
		} else {
			repo, err := opts.BaseRepo()
			if err != nil {
				return err
			}
			options := &gitlab.ListProjectAccessTokensOptions{PerPage: 100}
			tokens, err := api.ListProjectAccessTokens(httpClient, repo.FullName(), options)
			if err != nil {
				return err
			}
			tokens = filter.Filter(tokens, func(t *gitlab.ProjectAccessToken) bool {
				return t.Active && (t.Name == opts.Name || t.ID == opts.TokenId)
			})
			var token *gitlab.ProjectAccessToken
			switch len(tokens) {
			case 1:
				token = tokens[0]
			case 0:
				return cmdutils.FlagError{Err: fmt.Errorf("no token found with the name '%v'.", opts.Name)}
			default:
				return cmdutils.FlagError{Err: fmt.Errorf("multiple tokens found with the name '%v'. Use the ID instead.", opts.Name)}
			}

			if err = api.RevokeProjectAccessToken(httpClient, repo.FullName(), token.ID); err != nil {
				return err
			}
			token.Revoked = true
			outputToken = token
			outputTokenValue = fmt.Sprintf("revoked %s %d", token.Name, token.ID)
		}
	}

	if opts.OutputFormat == "json" {
		encoder := json.NewEncoder(opts.IO.StdOut)
		if err := encoder.Encode(outputToken); err != nil {
			return err
		}
	} else {
		if _, err := opts.IO.StdOut.Write([]byte(outputTokenValue)); err != nil {
			return err
		}
	}

	return nil
}
