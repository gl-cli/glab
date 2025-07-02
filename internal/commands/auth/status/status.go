package status

import (
	"fmt"
	"slices"
	"strings"

	"github.com/MakeNowJust/heredoc/v2"
	"github.com/spf13/cobra"
	"gitlab.com/gitlab-org/cli/internal/api"
	"gitlab.com/gitlab-org/cli/internal/cmdutils"
	"gitlab.com/gitlab-org/cli/internal/config"
	"gitlab.com/gitlab-org/cli/internal/glinstance"
	"gitlab.com/gitlab-org/cli/internal/iostreams"
)

type options struct {
	hostname  string
	showToken bool

	httpClientOverride func(token, hostname string) (*api.Client, error) // used in tests to mock http client
	io                 *iostreams.IOStreams
	apiClient          func(repoHost string, cfg config.Config) (*api.Client, error)
	config             func() config.Config
}

func NewCmdStatus(f cmdutils.Factory, runE func(*options) error) *cobra.Command {
	opts := &options{
		io:        f.IO(),
		apiClient: f.ApiClient,
		config:    f.Config,
	}

	cmd := &cobra.Command{
		Use:   "status",
		Args:  cobra.ExactArgs(0),
		Short: "View authentication status.",
		Long: heredoc.Doc(`Verifies and displays information about your authentication state.

		This command tests the authentication states of all known GitLab instances in the configuration file and reports issues, if any.
		`),
		RunE: func(cmd *cobra.Command, args []string) error {
			if runE != nil {
				return runE(opts)
			}

			return opts.run()
		},
	}

	cmd.Flags().StringVarP(&opts.hostname, "hostname", "h", "", "Check a specific instance's authentication status.")
	cmd.Flags().BoolVarP(&opts.showToken, "show-token", "t", false, "Display the authentication token.")

	return cmd
}

func (o *options) run() error {
	c := o.io.Color()
	cfg := o.config()

	stderr := o.io.StdErr

	statusInfo := map[string][]string{}

	instances, err := cfg.Hosts()
	if len(instances) == 0 || err != nil {
		return fmt.Errorf("No GitLab instances have been authenticated with glab. Run `%s` to authenticate.\n", c.Bold("glab auth login"))
	}

	if o.hostname != "" && !slices.Contains(instances, o.hostname) {
		return fmt.Errorf("%s %s has not been authenticated with glab. Run `%s %s` to authenticate.", c.FailedIcon(), o.hostname, c.Bold("glab auth login --hostname"), c.Bold(o.hostname))
	}

	failedAuth := false
	for _, instance := range instances {
		if o.hostname != "" && o.hostname != instance {
			continue
		}
		statusInfo[instance] = []string{}
		addMsg := func(x string, ys ...any) {
			statusInfo[instance] = append(statusInfo[instance], fmt.Sprintf(x, ys...))
		}

		token, tokenSource, _ := cfg.GetWithSource(instance, "token", false)
		apiClient, err := o.apiClient(instance, cfg)
		if o.httpClientOverride != nil {
			apiClient, _ = o.httpClientOverride(token, instance)
		}
		if err == nil {
			user, _, err := apiClient.Lab().Users.CurrentUser()
			if err != nil {
				failedAuth = true
				addMsg("%s %s: API call failed: %s", c.FailedIcon(), instance, err)
			} else {
				addMsg("%s Logged in to %s as %s (%s)", c.GreenCheck(), instance, c.Bold(user.Username), tokenSource)
			}
		} else {
			failedAuth = true
			addMsg("%s %s: failed to initialize api client: %s", c.FailedIcon(), instance, err)
		}
		proto, _ := cfg.Get(instance, "git_protocol")
		if proto != "" {
			addMsg("%s Git operations for %s configured to use %s protocol.",
				c.GreenCheck(), instance, c.Bold(proto))
		}
		apiProto, _ := cfg.Get(instance, "api_protocol")
		apiHost, _ := cfg.Get(instance, "api_host")
		apiEndpoint := glinstance.APIEndpoint(instance, apiProto, apiHost)
		graphQLEndpoint := glinstance.GraphQLEndpoint(instance, apiProto)
		if apiProto != "" {
			addMsg("%s API calls for %s are made over %s protocol.",
				c.GreenCheck(), instance, c.Bold(apiProto))
			addMsg("%s REST API Endpoint: %s",
				c.GreenCheck(), c.Bold(apiEndpoint))
			addMsg("%s GraphQL Endpoint: %s",
				c.GreenCheck(), c.Bold(graphQLEndpoint))
		}
		if token != "" {
			tokenDisplay := "**************************"
			if o.showToken {
				tokenDisplay = token
			}
			addMsg("%s Token: %s", c.GreenCheck(), tokenDisplay)
			if !api.IsValidToken(token) {
				addMsg("%s Invalid token provided in configuration file.", c.WarnIcon())
			}
		} else {
			addMsg("%s No token provided in configuration file.", c.WarnIcon())
		}
	}

	for _, instance := range instances {
		if o.hostname != "" && o.hostname != instance {
			continue
		}

		lines, ok := statusInfo[instance]
		if !ok {
			continue
		}
		fmt.Fprintf(stderr, "%s\n", c.Bold(instance))
		for _, line := range lines {
			fmt.Fprintf(stderr, "  %s\n", line)
		}
	}

	envToken := config.GetFromEnv("token")
	if envToken != "" {
		fmt.Fprintf(stderr, "\n%s One of %s environment variables is set. It will be used for all authentication.\n", c.WarnIcon(), strings.Join(config.EnvKeyEquivalence("token"), ", "))
	}

	if failedAuth {
		return fmt.Errorf("\n%s could not authenticate to one or more of the configured GitLab instances.", c.FailedIcon())
	} else {
		return nil
	}
}
