package rotate

import (
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"time"

	"gitlab.com/gitlab-org/cli/commands/token/expirationdate"
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

type RotateOptions struct {
	HTTPClient func() (*gitlab.Client, error)
	IO         *iostreams.IOStreams
	BaseRepo   func() (glrepo.Interface, error)

	User         string
	Group        string
	Name         interface{}
	Duration     time.Duration
	ExpireAt     expirationdate.ExpirationDate
	OutputFormat string
}

func NewCmdRotate(f *cmdutils.Factory, runE func(opts *RotateOptions) error) *cobra.Command {
	opts := &RotateOptions{
		IO: f.IO,
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
			# Rotate project access token of current project
			glab token rotate  my-project-token

			# Rotate project access token of another project, set to expiration date
			glab token rotate --repo user/repo my-project-token --expires-at 2024-08-08

			# Rotate group access token
			glab token rotate --group group/sub-group my-group-token

			# Rotate personal access token and extend duration to 7 days
			glab token rotate --user @me --duration $((7 * 24))h my-personal-token

			# Rotate a personal access token of another user (administrator only)
			glab token rotate --user johndoe johns-personal-token
		`),
		RunE: func(cmd *cobra.Command, args []string) (err error) {
			// Supports repo override
			opts.HTTPClient = f.HttpClient
			opts.BaseRepo = f.BaseRepo

			if opts.Name, err = strconv.Atoi(args[0]); err != nil {
				opts.Name = args[0]
			}
			if opts.Group, err = flag.GroupOverride(cmd); err != nil {
				return
			}

			if opts.Group != "" && opts.User != "" {
				return cmdutils.FlagError{Err: errors.New("'--group' and '--user' are mutually exclusive.")}
			}

			if opts.Duration.Truncate(24*time.Hour) != opts.Duration {
				return cmdutils.FlagError{Err: errors.New("duration must be in days.")}
			}

			if opts.Duration < 24*time.Hour || opts.Duration > 365*24*time.Hour {
				return cmdutils.FlagError{Err: errors.New("duration in days must be between 1 and 365.")}
			}

			if cmd.Flags().Changed("expires-at") && cmd.Flags().Changed("duration") {
				return cmdutils.FlagError{Err: errors.New("'--expires-at' and '--duration' are mutually exclusive.")}
			}

			if time.Time(opts.ExpireAt).IsZero() {
				opts.ExpireAt = expirationdate.ExpirationDate(time.Now().Add(opts.Duration).Truncate(time.Hour * 24))
			}

			if runE != nil {
				return runE(opts)
			}
			return rotateTokenRun(opts)
		},
	}

	cmdutils.EnableRepoOverride(cmd, f)
	cmd.Flags().StringVarP(&opts.Group, "group", "g", "", "Rotate group access token. Ignored if a user or repository argument is set.")
	cmd.Flags().StringVarP(&opts.User, "user", "U", "", "Rotate personal access token. Use @me for the current user.")
	cmd.Flags().DurationVarP(&opts.Duration, "duration", "D", time.Duration(30*24*time.Hour), "Sets the token duration, in hours. Maximum of 8760. Examples: 24h, 168h, 504h.")
	cmd.Flags().VarP(&opts.ExpireAt, "expires-at", "E", "Sets the token's expiration date and time, in YYYY-MM-DD format. If not specified, --duration is used.")
	cmd.Flags().StringVarP(&opts.OutputFormat, "output", "F", "text", "Format output as: text, json. 'text' provides the new token value; 'json' outputs the token with metadata.")
	return cmd
}

func rotateTokenRun(opts *RotateOptions) error {
	httpClient, err := opts.HTTPClient()
	if err != nil {
		return err
	}
	expirationDate := gitlab.ISOTime(opts.ExpireAt)

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
			return t.Active && (t.Name == opts.Name || t.ID == opts.Name)
		})
		switch len(tokens) {
		case 1:
			token = tokens[0]
		case 0:
			return cmdutils.FlagError{Err: fmt.Errorf("no token found with the name '%v'", opts.Name)}
		default:
			return cmdutils.FlagError{Err: fmt.Errorf("multiple tokens found with the name '%v'. Use the ID instead.", opts.Name)}
		}
		rotateOptions := &gitlab.RotatePersonalAccessTokenOptions{
			ExpiresAt: &expirationDate,
		}
		if token, err = api.RotatePersonalAccessToken(httpClient, token.ID, rotateOptions); err != nil {
			return err
		}
		outputToken = token
		outputTokenValue = token.Token
	} else {
		if opts.Group != "" {
			options := &gitlab.ListGroupAccessTokensOptions{PerPage: 100}
			tokens, err := api.ListGroupAccessTokens(httpClient, opts.Group, options)
			if err != nil {
				return err
			}
			var token *gitlab.GroupAccessToken
			tokens = filter.Filter(tokens, func(t *gitlab.GroupAccessToken) bool {
				return t.Active && (t.Name == opts.Name || t.ID == opts.Name)
			})
			switch len(tokens) {
			case 1:
				token = tokens[0]
			case 0:
				return cmdutils.FlagError{Err: fmt.Errorf("no token found with the name '%v'", opts.Name)}
			default:
				return cmdutils.FlagError{Err: fmt.Errorf("multiple tokens found with the name '%v', use the ID instead", opts.Name)}
			}

			rotateOptions := &gitlab.RotateGroupAccessTokenOptions{
				ExpiresAt: &expirationDate,
			}
			if token, err = api.RotateGroupAccessToken(httpClient, opts.Group, token.ID, rotateOptions); err != nil {
				return err
			}
			outputToken = token
			outputTokenValue = token.Token
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
				return t.Active && (t.Name == opts.Name || t.ID == opts.Name)
			})
			var token *gitlab.ProjectAccessToken
			switch len(tokens) {
			case 1:
				token = tokens[0]
			case 0:
				return cmdutils.FlagError{Err: fmt.Errorf("no token found with the name '%v'", opts.Name)}
			default:
				return cmdutils.FlagError{Err: fmt.Errorf("multiple tokens found with the name '%v', use the ID instead", opts.Name)}
			}

			rotateOptions := &gitlab.RotateProjectAccessTokenOptions{
				ExpiresAt: &expirationDate,
			}
			if token, err = api.RotateProjectAccessToken(httpClient, repo.FullName(), token.ID, rotateOptions); err != nil {
				return err
			}
			outputToken = token
			outputTokenValue = token.Token
		}
	}

	if opts.OutputFormat == "json" {
		encoder := json.NewEncoder(opts.IO.StdOut)
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
