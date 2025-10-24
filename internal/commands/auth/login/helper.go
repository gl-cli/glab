package login

import (
	"bufio"
	"fmt"
	"net/url"
	"strings"

	gitlab "gitlab.com/gitlab-org/api/client-go"
	"gitlab.com/gitlab-org/cli/internal/api"
	"gitlab.com/gitlab-org/cli/internal/cmdutils"
	"gitlab.com/gitlab-org/cli/internal/config"
	"gitlab.com/gitlab-org/cli/internal/iostreams"
	"gitlab.com/gitlab-org/cli/internal/mcpannotations"

	"github.com/spf13/cobra"
)

type options struct {
	io        *iostreams.IOStreams
	config    func() config.Config
	apiClient func(repoHost string) (*api.Client, error)

	operation string
}

func NewCmdCredential(f cmdutils.Factory) *cobra.Command {
	opts := &options{
		io:        f.IO(),
		config:    f.Config,
		apiClient: f.ApiClient,
	}

	cmd := &cobra.Command{
		Use:    "git-credential",
		Args:   cobra.ExactArgs(1),
		Short:  "Implements Git credential helper manager.",
		Hidden: true,
		Annotations: map[string]string{
			mcpannotations.Destructive: "true",
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.complete(args)

			if err := opts.validate(); err != nil {
				return err
			}

			return opts.run()
		},
	}

	return cmd
}

func (o *options) complete(args []string) {
	o.operation = args[0]
}

func (o *options) validate() error {
	if o.operation != "get" {
		// Ignore unsupported operation
		return cmdutils.SilentError
	}
	return nil
}

func (o *options) run() error {
	expectedParams := map[string]string{}

	s := bufio.NewScanner(o.io.In)
	for s.Scan() {
		line := s.Text()
		if line == "" {
			break
		}
		parts := strings.SplitN(line, "=", 2)
		if len(parts) < 2 {
			continue
		}
		key, value := parts[0], parts[1]
		if key == "url" {
			u, err := url.Parse(value)
			if err != nil {
				return err
			}
			expectedParams["protocol"] = u.Scheme
			expectedParams["host"] = u.Host
			expectedParams["path"] = u.Path
			expectedParams["username"] = u.User.Username()
			expectedParams["password"], _ = u.User.Password()
		} else {
			expectedParams[key] = value
		}
	}
	if err := s.Err(); err != nil {
		return err
	}

	if expectedParams["protocol"] != "https" && expectedParams["protocol"] != "http" {
		return cmdutils.SilentError
	}

	cfg := o.config()

	output := map[string]string{}

	host := expectedParams["host"]
	isOAuth2Cfg, _ := cfg.Get(host, "is_oauth2")
	jobToken, _ := cfg.Get(host, "job_token")
	token, _ := cfg.Get(host, "token")
	username, _ := cfg.Get(host, "user")

	switch {
	case isOAuth2Cfg == "true":
		// Trying to refresh access token
		apiClient, err := o.apiClient(host)
		if err != nil {
			return err
		}
		// The AuthSource for apiClient with OAuth2 settings should gives back
		// gitlab.OAuthTokenSource, which should pass type assertion here.
		authSource := apiClient.AuthSource().(gitlab.OAuthTokenSource)
		oauth2Token, err := authSource.TokenSource.Token()
		if err != nil {
			return fmt.Errorf("failed to refresh token for %q: %w", host, err)
		}

		// see https://docs.gitlab.com/ee/api/oauth2.html#access-git-over-https-with-access-token
		output["username"] = "oauth2"
		output["password"] = oauth2Token.AccessToken
		if !oauth2Token.Expiry.IsZero() {
			output["password_expiry_utc"] = fmt.Sprintf("%d", oauth2Token.Expiry.UTC().Unix())
		}
		if oauth2Token.RefreshToken != "" {
			output["oauth_refresh_token"] = oauth2Token.RefreshToken
		}
	case jobToken != "":
		output["username"] = "gitlab-ci-token"
		// see https://docs.gitlab.com/ci/jobs/ci_job_token/#to-git-clone-a-private-projects-repository
		output["password"] = jobToken
	case token != "":
		output["username"] = username
		output["password"] = token
	default:
		return cmdutils.SilentError
	}

	if expectedParams["username"] != "" && expectedParams["username"] != output["username"] {
		return fmt.Errorf("the requested username by Git doesn't match the one that is configured for this host with GLab, want %q but got %q. Rejecting request", output["username"], expectedParams["username"])
	}

	// "A capability[] directive must precede any value depending on it and these directives should be the first item announced in the protocol." https://git-scm.com/docs/git-credential
	fmt.Fprintln(o.io.StdOut, "capability[]=authtype")
	for key, v := range output {
		fmt.Fprintf(o.io.StdOut, "%s=%s\n", key, v)
	}

	return nil
}
