package rotate

import (
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"time"

	"gitlab.com/gitlab-org/cli/internal/commands/token/expirationdate"
	"gitlab.com/gitlab-org/cli/internal/commands/token/filter"

	"gitlab.com/gitlab-org/cli/internal/iostreams"

	"github.com/MakeNowJust/heredoc/v2"
	"github.com/spf13/cobra"
	gitlab "gitlab.com/gitlab-org/api/client-go"
	"gitlab.com/gitlab-org/cli/internal/api"
	"gitlab.com/gitlab-org/cli/internal/cmdutils"
	"gitlab.com/gitlab-org/cli/internal/config"
	"gitlab.com/gitlab-org/cli/internal/glrepo"
)

type options struct {
	apiClient func(repoHost string, cfg config.Config) (*api.Client, error)
	config    config.Config
	io        *iostreams.IOStreams
	baseRepo  func() (glrepo.Interface, error)

	user         string
	group        string
	name         any
	duration     time.Duration
	expireAt     expirationdate.ExpirationDate
	outputFormat string
}

func NewCmdRotate(f cmdutils.Factory) *cobra.Command {
	opts := &options{
		io:        f.IO(),
		apiClient: f.ApiClient,
		config:    f.Config(),
		baseRepo:  f.BaseRepo,
	}

	cmd := &cobra.Command{
		Use:     "rotate <token-name|token-id>",
		Short:   "Rotate user, group, or project access tokens",
		Aliases: []string{"rotate", "rot"},
		Args:    cobra.RangeArgs(1, 1),
		Long: heredoc.Doc(`
			Rotate user, group, or project access token, then print the new token on stdout. If multiple tokens with
			the same name exist, you can specify the ID of the token.

			The expiration date of the token will be calculated by adding the duration (default 30 days) to the
			current date. Alternatively you can specify a different duration or an explicit end date.

			The output format can be either "JSON" or "text". The JSON output will show the meta information of the
			rotated token.

			Administrators can rotate personal access tokens belonging to other users.
		`),
		Example: heredoc.Doc(`
			Rotate project access token of current project
			- glab token rotate  my-project-token

			Rotate project access token of another project, set to expiration date
			- glab token rotate --repo user/repo my-project-token --expires-at 2024-08-08

			Rotate group access token
			- glab token rotate --group group/sub-group my-group-token

			Rotate personal access token and extend duration to 7 days
			- glab token rotate --user @me --duration $((7 * 24))h my-personal-token

			Rotate a personal access token of another user (administrator only)
			- glab token rotate --user johndoe johns-personal-token
		`),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := opts.complete(cmd, args); err != nil {
				return err
			}

			if err := opts.validate(); err != nil {
				return err
			}

			return opts.run()
		},
	}

	cmdutils.EnableRepoOverride(cmd, f)
	cmd.Flags().StringVarP(&opts.group, "group", "g", "", "Rotate group access token. Ignored if a user or repository argument is set.")
	cmd.Flags().StringVarP(&opts.user, "user", "U", "", "Rotate personal access token. Use @me for the current user.")
	cmd.Flags().DurationVarP(&opts.duration, "duration", "D", 30*24*time.Hour, "Sets the token duration, in hours. Maximum of 8760. Examples: 24h, 168h, 504h.")
	cmd.Flags().VarP(&opts.expireAt, "expires-at", "E", "Sets the token's expiration date and time, in YYYY-MM-DD format. If not specified, --duration is used.")
	cmd.Flags().StringVarP(&opts.outputFormat, "output", "F", "text", "Format output as: text, json. 'text' provides the new token value; 'json' outputs the token with metadata.")
	cmd.MarkFlagsMutuallyExclusive("duration", "expires-at")
	return cmd
}

func (o *options) complete(cmd *cobra.Command, args []string) error {
	if name, err := strconv.Atoi(args[0]); err != nil {
		o.name = args[0]
	} else {
		o.name = name
	}

	if group, err := cmdutils.GroupOverride(cmd); err != nil {
		return err
	} else {
		o.group = group
	}

	if time.Time(o.expireAt).IsZero() {
		o.expireAt = expirationdate.ExpirationDate(time.Now().Add(o.duration).Truncate(time.Hour * 24))
	}

	return nil
}

func (o *options) validate() error {
	if o.group != "" && o.user != "" {
		return cmdutils.FlagError{Err: errors.New("'--group' and '--user' are mutually exclusive.")}
	}

	if o.duration.Truncate(24*time.Hour) != o.duration {
		return cmdutils.FlagError{Err: errors.New("duration must be in days.")}
	}

	if o.duration < 24*time.Hour || o.duration > 365*24*time.Hour {
		return cmdutils.FlagError{Err: errors.New("duration in days must be between 1 and 365.")}
	}

	return nil
}

func (o *options) run() error {
	// NOTE: this command can not only be used for projects,
	// so we have to manually check for the base repo, it it doesn't exist,
	// we bootstrap the client with the default hostname.
	var repoHost string
	if baseRepo, err := o.baseRepo(); err == nil {
		repoHost = baseRepo.RepoHost()
	}
	apiClient, err := o.apiClient(repoHost, o.config)
	if err != nil {
		return err
	}
	client := apiClient.Lab()

	expirationDate := gitlab.ISOTime(o.expireAt)

	var outputToken any
	var outputTokenValue string

	if o.user != "" {
		user, err := api.UserByName(client, o.user)
		if err != nil {
			return cmdutils.FlagError{Err: err}
		}

		options := &gitlab.ListPersonalAccessTokensOptions{
			ListOptions: gitlab.ListOptions{PerPage: 100},
			UserID:      &user.ID,
		}
		tokens, err := gitlab.ScanAndCollect(func(p gitlab.PaginationOptionFunc) ([]*gitlab.PersonalAccessToken, *gitlab.Response, error) {
			return client.PersonalAccessTokens.ListPersonalAccessTokens(options, p)
		})
		if err != nil {
			return err
		}
		var token *gitlab.PersonalAccessToken
		tokens = filter.Filter(tokens, func(t *gitlab.PersonalAccessToken) bool {
			return t.Active && (t.Name == o.name || t.ID == o.name)
		})
		switch len(tokens) {
		case 1:
			token = tokens[0]
		case 0:
			return cmdutils.FlagError{Err: fmt.Errorf("no token found with the name '%v'", o.name)}
		default:
			return cmdutils.FlagError{Err: fmt.Errorf("multiple tokens found with the name '%v'. Use the ID instead.", o.name)}
		}
		rotateOptions := &gitlab.RotatePersonalAccessTokenOptions{
			ExpiresAt: &expirationDate,
		}
		if token, _, err = client.PersonalAccessTokens.RotatePersonalAccessToken(token.ID, rotateOptions); err != nil {
			return err
		}
		outputToken = token
		outputTokenValue = token.Token
	} else {
		if o.group != "" {
			options := &gitlab.ListGroupAccessTokensOptions{ListOptions: gitlab.ListOptions{PerPage: 100}}
			tokens, err := gitlab.ScanAndCollect(func(p gitlab.PaginationOptionFunc) ([]*gitlab.GroupAccessToken, *gitlab.Response, error) {
				return client.GroupAccessTokens.ListGroupAccessTokens(o.group, options, p)
			})
			if err != nil {
				return err
			}
			var token *gitlab.GroupAccessToken
			tokens = filter.Filter(tokens, func(t *gitlab.GroupAccessToken) bool {
				return t.Active && (t.Name == o.name || t.ID == o.name)
			})
			switch len(tokens) {
			case 1:
				token = tokens[0]
			case 0:
				return cmdutils.FlagError{Err: fmt.Errorf("no token found with the name '%v'", o.name)}
			default:
				return cmdutils.FlagError{Err: fmt.Errorf("multiple tokens found with the name '%v', use the ID instead", o.name)}
			}

			rotateOptions := &gitlab.RotateGroupAccessTokenOptions{
				ExpiresAt: &expirationDate,
			}
			if token, _, err = client.GroupAccessTokens.RotateGroupAccessToken(o.group, token.ID, rotateOptions); err != nil {
				return err
			}
			outputToken = token
			outputTokenValue = token.Token
		} else {
			repo, err := o.baseRepo()
			if err != nil {
				return err
			}
			options := &gitlab.ListProjectAccessTokensOptions{ListOptions: gitlab.ListOptions{PerPage: 100}}
			tokens, err := gitlab.ScanAndCollect(func(p gitlab.PaginationOptionFunc) ([]*gitlab.ProjectAccessToken, *gitlab.Response, error) {
				return client.ProjectAccessTokens.ListProjectAccessTokens(repo.FullName(), options, p)
			})
			if err != nil {
				return err
			}
			tokens = filter.Filter(tokens, func(t *gitlab.ProjectAccessToken) bool {
				return t.Active && (t.Name == o.name || t.ID == o.name)
			})
			var token *gitlab.ProjectAccessToken
			switch len(tokens) {
			case 1:
				token = tokens[0]
			case 0:
				return cmdutils.FlagError{Err: fmt.Errorf("no token found with the name '%v'", o.name)}
			default:
				return cmdutils.FlagError{Err: fmt.Errorf("multiple tokens found with the name '%v', use the ID instead", o.name)}
			}

			rotateOptions := &gitlab.RotateProjectAccessTokenOptions{
				ExpiresAt: &expirationDate,
			}
			if token, _, err = client.ProjectAccessTokens.RotateProjectAccessToken(repo.FullName(), token.ID, rotateOptions); err != nil {
				return err
			}
			outputToken = token
			outputTokenValue = token.Token
		}
	}

	if o.outputFormat == "json" {
		encoder := json.NewEncoder(o.io.StdOut)
		if err := encoder.Encode(outputToken); err != nil {
			return err
		}
	} else {
		if _, err := fmt.Fprintf(o.io.StdOut, "%s\n", outputTokenValue); err != nil {
			return err
		}
	}

	return nil
}
