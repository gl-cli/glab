package create

import (
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"gitlab.com/gitlab-org/cli/commands/token/expirationdate"
	"gitlab.com/gitlab-org/cli/commands/token/filter"

	"gitlab.com/gitlab-org/cli/commands/flag"
	"gitlab.com/gitlab-org/cli/commands/token/accesslevel"

	"gitlab.com/gitlab-org/cli/pkg/iostreams"

	"github.com/MakeNowJust/heredoc/v2"
	"github.com/spf13/cobra"
	"github.com/xanzy/go-gitlab"
	"gitlab.com/gitlab-org/cli/api"
	"gitlab.com/gitlab-org/cli/commands/cmdutils"
	"gitlab.com/gitlab-org/cli/internal/glrepo"
)

type CreateOptions struct {
	HTTPClient func() (*gitlab.Client, error)
	IO         *iostreams.IOStreams
	BaseRepo   func() (glrepo.Interface, error)

	Name         string
	User         string
	Group        string
	AccessLevel  accesslevel.AccessLevel
	Scopes       []string
	Duration     time.Duration
	ExpireAt     expirationdate.ExpirationDate
	OutputFormat string
}

func NewCmdCreate(f *cmdutils.Factory, runE func(opts *CreateOptions) error) *cobra.Command {
	opts := &CreateOptions{
		IO: f.IO,
	}

	cmd := &cobra.Command{
		Use:     "create <name>",
		Aliases: []string{"create", "new"},
		Args:    cobra.RangeArgs(1, 1),
		Short:   "Creates user, group, or project access tokens.",
		Long: heredoc.Doc(`
		Creates a new access token for a user, group, or project. Defaults to a
		project access token, unless user or group name is specified.

		The expiration date of the token is calculated by adding the duration
		(default: 30 days) to the current date. You can specify a different duration,
		or an explicit end date.

		The name of the token must be unique. The token is printed to stdout.

		Administrators can create full-featured personal access tokens for themselves and for other users.
		Non-administrators can create personal access tokens only for themselves (@me) with the scope 'k8s_proxy'.
		`),
		Example: heredoc.Doc(`
		# Create project access token for current project
		glab token create --access-level developer --scope read_repository --scope read_registry my-project-token

		# Create project access token for a specific project
		glab token create --repo user/my-repo --access-level owner --scope api my-project-token

		# Create a group access token
		glab token create --group group/sub-group --access-level owner --scope api my-group-token

		# Create a personal access token for current user
		glab token create --user @me --scope k8s_proxy my-personal-token

		# (administrator only) Create a personal access token for another user
		glab token create --user johndoe --scope api johns-personal-token

		`),
		RunE: func(cmd *cobra.Command, args []string) (err error) {
			// Supports repo override
			opts.HTTPClient = f.HttpClient
			opts.BaseRepo = f.BaseRepo
			opts.Name = args[0]
			if opts.Group, err = flag.GroupOverride(cmd); err != nil {
				return
			}

			if opts.Group != "" && opts.User != "" {
				return cmdutils.FlagError{Err: errors.New("'--group' and '--user' are mutually exclusive.")}
			}

			if cmd.Flags().Changed("expires-at") && cmd.Flags().Changed("duration") {
				return cmdutils.FlagError{Err: errors.New("'--expires-at' and '--duration' are mutually exclusive.")}
			}

			if time.Time(opts.ExpireAt).IsZero() {
				opts.ExpireAt = expirationdate.ExpirationDate(time.Now().Add(opts.Duration).Truncate(time.Hour * 24))
			}

			if opts.Duration.Truncate(24*time.Hour) != opts.Duration {
				return cmdutils.FlagError{Err: errors.New("duration must be in days.")}
			}

			if opts.Duration < 24*time.Hour || opts.Duration > 365*24*time.Hour {
				return cmdutils.FlagError{Err: errors.New("duration in days must be between 1 and 365.")}
			}

			if opts.User != "" && opts.Group != "" {
				return cmdutils.FlagError{Err: errors.New("'--user' and '--group' are mutually exclusive.")}
			}

			if opts.User == "" && !cmd.Flag("access-level").Changed {
				return cmdutils.FlagError{Err: errors.New("the required flag '--access-level' is not set.")}
			}

			if runE != nil {
				err = runE(opts)
				return
			}
			err = createTokenRun(opts)
			return
		},
	}

	cmdutils.EnableRepoOverride(cmd, f)
	cmd.Flags().StringVarP(&opts.Group, "group", "g", "", "Create a group access token. Ignored if a user or repository argument is set.")
	cmd.Flags().StringVarP(&opts.User, "user", "U", "", "Create a personal access token. For the current user, use @me.")
	cmd.Flags().DurationVarP(&opts.Duration, "duration", "D", time.Duration(30*24*time.Hour), "Sets the token duration, in hours. Maximum of 8760. Examples: 24h, 168h, 504h.")
	cmd.Flags().VarP(&opts.ExpireAt, "expires-at", "E", "Sets the token's expiration date and time, in YYYY-MM-DD format. If not specified, --duration is used.")
	cmd.Flags().StringSliceVarP(&opts.Scopes, "scope", "S", []string{"read_repository"}, "Scopes for the token. For a list, see https://docs.gitlab.com/ee/user/profile/personal_access_tokens.html#personal-access-token-scopes.")
	cmd.Flags().VarP(&opts.AccessLevel, "access-level", "A", "Access level of the token: one of 'guest', 'reporter', 'developer', 'maintainer', 'owner'.")
	cmd.Flags().StringVarP(&opts.OutputFormat, "output", "F", "text", "Format output as 'text' for the token value, 'json' for the actual API token structure.")
	return cmd
}

func createTokenRun(opts *CreateOptions) error {
	httpClient, err := opts.HTTPClient()
	if err != nil {
		return err
	}
	var outputToken interface{}
	var outputTokenValue string
	expirationDate := gitlab.ISOTime(opts.ExpireAt)

	if opts.User != "" {
		user, err := api.UserByName(httpClient, opts.User)
		if err != nil {
			return err
		}

		options := &gitlab.ListPersonalAccessTokensOptions{
			ListOptions: gitlab.ListOptions{PerPage: 100},
			UserID:      &user.ID,
		}
		tokens, err := api.ListPersonalAccessTokens(httpClient, options)
		if err != nil {
			return err
		}
		tokens = filter.Filter(tokens, func(t *gitlab.PersonalAccessToken) bool {
			return t.Active && t.Name == opts.Name
		})
		if len(tokens) > 0 {
			return cmdutils.FlagError{Err: fmt.Errorf("a personal access token with the name %s already exists.", opts.Name)}
		}

		if opts.User == "@me" {
			token, err := api.CreatePersonalAccessTokenForCurrentUser(httpClient, opts.Name, opts.Scopes, time.Time(expirationDate))
			if err != nil {
				return err
			}
			outputToken = token
			outputTokenValue = token.Token
		} else {
			createOptions := &gitlab.CreatePersonalAccessTokenOptions{
				Name:      &opts.Name,
				ExpiresAt: &expirationDate,
				Scopes:    &opts.Scopes,
			}
			token, err := api.CreatePersonalAccessToken(httpClient, user.ID, createOptions)
			if err != nil {
				return err
			}
			outputToken = token
			outputTokenValue = token.Token
		}
	} else {
		if opts.Group != "" {
			listOptions := &gitlab.ListGroupAccessTokensOptions{PerPage: 100}
			tokens, err := api.ListGroupAccessTokens(httpClient, opts.Group, listOptions)
			if err != nil {
				return err
			}
			tokens = filter.Filter(tokens, func(t *gitlab.GroupAccessToken) bool {
				return t.Active && t.Name == opts.Name
			})
			if len(tokens) > 0 {
				return cmdutils.FlagError{Err: fmt.Errorf("a group access token with the name %s already exists.", opts.Name)}
			}

			options := gitlab.CreateGroupAccessTokenOptions{
				Name:        &opts.Name,
				Scopes:      &opts.Scopes,
				AccessLevel: &opts.AccessLevel.Value,
				ExpiresAt:   &expirationDate,
			}
			token, err := api.CreateGroupAccessToken(httpClient, opts.Group, &options)
			if err != nil {
				return err
			}
			outputToken = token
			outputTokenValue = token.Token

		} else {
			repo, err := opts.BaseRepo()
			if err != nil {
				return err
			}
			listOptions := &gitlab.ListProjectAccessTokensOptions{PerPage: 100}
			tokens, err := api.ListProjectAccessTokens(httpClient, repo.FullName(), listOptions)
			if err != nil {
				return err
			}
			tokens = filter.Filter(tokens, func(t *gitlab.ProjectAccessToken) bool {
				return t.Active && t.Name == opts.Name
			})

			if len(tokens) > 0 {
				return cmdutils.FlagError{Err: fmt.Errorf("a project access token with name %s already exists.", opts.Name)}
			}

			options := gitlab.CreateProjectAccessTokenOptions{
				Name:        &opts.Name,
				Scopes:      &opts.Scopes,
				AccessLevel: &opts.AccessLevel.Value,
				ExpiresAt:   &expirationDate,
			}
			token, err := api.CreateProjectAccessToken(httpClient, repo.FullName(), &options)
			if err != nil {
				return err
			}
			outputToken = token
			outputTokenValue = token.Token
		}
	}

	if opts.OutputFormat == "json" {
		encoder := json.NewEncoder(opts.IO.StdOut)
		encoder.SetIndent("  ", "  ")
		if err := encoder.Encode(outputToken); err != nil {
			return err
		}
	} else {
		if _, err := fmt.Fprintf(opts.IO.StdOut, "%s\n", outputTokenValue); err != nil {
			return err
		}
	}

	return nil
}
