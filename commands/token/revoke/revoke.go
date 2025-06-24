package revoke

import (
	"encoding/json"
	"fmt"
	"strconv"

	"gitlab.com/gitlab-org/cli/commands/token/filter"

	"gitlab.com/gitlab-org/cli/commands/flag"
	"gitlab.com/gitlab-org/cli/pkg/iostreams"

	"github.com/MakeNowJust/heredoc/v2"
	"github.com/spf13/cobra"
	gitlab "gitlab.com/gitlab-org/api/client-go"
	"gitlab.com/gitlab-org/cli/api"
	"gitlab.com/gitlab-org/cli/commands/cmdutils"
	"gitlab.com/gitlab-org/cli/internal/glrepo"
)

type options struct {
	httpClient func() (*gitlab.Client, error)
	io         *iostreams.IOStreams
	baseRepo   func() (glrepo.Interface, error)

	user         string
	group        string
	name         string
	tokenID      int
	outputFormat string
}

func NewCmdRevoke(f cmdutils.Factory) *cobra.Command {
	opts := &options{
		io:         f.IO(),
		httpClient: f.HttpClient,
		baseRepo:   f.BaseRepo,
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
			Revoke a project access token of current project
			- glab token revoke my-project-token

			Revoke a project access token of a specific project
			- glab token revoke --repo user/my-repo my-project-token

			Revoke a group access token
			- glab token revoke --group group/sub-group my-group-token

			Revoke my personal access token
			- glab token revoke --user @me my-personal-token

			Revoke a personal access token of another user (administrator only)
			- glab token revoke --user johndoe johns-personal-token

		`),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := opts.complete(cmd, args); err != nil {
				return err
			}

			return opts.run()
		},
	}

	cmdutils.EnableRepoOverride(cmd, f)
	cmd.Flags().StringVarP(&opts.group, "group", "g", "", "Revoke group access token. Ignored if a user or repository argument is set.")
	cmd.Flags().StringVarP(&opts.user, "user", "U", "", "Revoke personal access token. Use @me for the current user.")
	cmd.Flags().StringVarP(&opts.outputFormat, "output", "F", "text", "Format output as: text, json. 'text' provides the name and ID of the revoked token; 'json' outputs the token with metadata.")
	cmd.MarkFlagsMutuallyExclusive("group", "user")
	return cmd
}

func (o *options) complete(cmd *cobra.Command, args []string) error {
	if tokenID, err := strconv.Atoi(args[0]); err != nil {
		o.name = args[0]
	} else {
		o.tokenID = tokenID
	}
	if group, err := flag.GroupOverride(cmd); err != nil {
		return err
	} else {
		o.group = group
	}

	return nil
}

func (o *options) run() error {
	httpClient, err := o.httpClient()
	if err != nil {
		return err
	}

	var outputToken any
	var outputTokenValue string

	if o.user != "" {
		user, err := api.UserByName(httpClient, o.user)
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
			return t.Active && (t.Name == o.name || t.ID == o.tokenID)
		})
		switch len(tokens) {
		case 1:
			token = tokens[0]
		case 0:
			return cmdutils.FlagError{Err: fmt.Errorf("no token found with the name '%v'.", o.name)}
		default:
			return cmdutils.FlagError{Err: fmt.Errorf("multiple tokens found with the name '%v'. Use the ID instead.", o.name)}
		}
		if err = api.RevokePersonalAccessToken(httpClient, token.ID); err != nil {
			return err
		}
		token.Revoked = true
		outputToken = token
		outputTokenValue = fmt.Sprintf("revoked %s %s %d", o.user, token.Name, token.ID)
	} else {
		if o.group != "" {
			options := &gitlab.ListGroupAccessTokensOptions{ListOptions: gitlab.ListOptions{PerPage: 100}}
			tokens, err := api.ListGroupAccessTokens(httpClient, o.group, options)
			if err != nil {
				return err
			}
			var token *gitlab.GroupAccessToken
			tokens = filter.Filter(tokens, func(t *gitlab.GroupAccessToken) bool {
				return t.Active && (t.Name == o.name || t.ID == o.tokenID)
			})
			switch len(tokens) {
			case 1:
				token = tokens[0]
			case 0:
				return cmdutils.FlagError{Err: fmt.Errorf("no token found with the name '%v'.", o.name)}
			default:
				return cmdutils.FlagError{Err: fmt.Errorf("multiple tokens found with the name '%v'. Use the ID instead.", o.name)}
			}

			if err = api.RevokeGroupAccessToken(httpClient, o.group, token.ID); err != nil {
				return err
			}
			token.Revoked = true
			outputToken = token
			outputTokenValue = fmt.Sprintf("revoked %s %d", token.Name, token.ID)
		} else {
			repo, err := o.baseRepo()
			if err != nil {
				return err
			}
			options := &gitlab.ListProjectAccessTokensOptions{ListOptions: gitlab.ListOptions{PerPage: 100}}
			tokens, err := api.ListProjectAccessTokens(httpClient, repo.FullName(), options)
			if err != nil {
				return err
			}
			tokens = filter.Filter(tokens, func(t *gitlab.ProjectAccessToken) bool {
				return t.Active && (t.Name == o.name || t.ID == o.tokenID)
			})
			var token *gitlab.ProjectAccessToken
			switch len(tokens) {
			case 1:
				token = tokens[0]
			case 0:
				return cmdutils.FlagError{Err: fmt.Errorf("no token found with the name '%v'.", o.name)}
			default:
				return cmdutils.FlagError{Err: fmt.Errorf("multiple tokens found with the name '%v'. Use the ID instead.", o.name)}
			}

			if err = api.RevokeProjectAccessToken(httpClient, repo.FullName(), token.ID); err != nil {
				return err
			}
			token.Revoked = true
			outputToken = token
			outputTokenValue = fmt.Sprintf("revoked %s %d", token.Name, token.ID)
		}
	}

	if o.outputFormat == "json" {
		encoder := json.NewEncoder(o.io.StdOut)
		if err := encoder.Encode(outputToken); err != nil {
			return err
		}
	} else {
		if _, err := o.io.StdOut.Write([]byte(outputTokenValue)); err != nil {
			return err
		}
	}

	return nil
}
