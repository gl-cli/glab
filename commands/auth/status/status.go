package status

import (
	"fmt"
	"strings"

	"golang.org/x/exp/slices"

	"gitlab.com/gitlab-org/cli/pkg/iostreams"

	"github.com/MakeNowJust/heredoc/v2"
	"github.com/spf13/cobra"
	"gitlab.com/gitlab-org/cli/api"
	"gitlab.com/gitlab-org/cli/commands/cmdutils"
	"gitlab.com/gitlab-org/cli/internal/config"
	"gitlab.com/gitlab-org/cli/pkg/glinstance"
)

type StatusOpts struct {
	Hostname  string
	ShowToken bool

	HttpClientOverride func(token, hostname string) (*api.Client, error) // used in tests to mock http client
	IO                 *iostreams.IOStreams
	Config             func() (config.Config, error)
}

func NewCmdStatus(f *cmdutils.Factory, runE func(*StatusOpts) error) *cobra.Command {
	opts := &StatusOpts{
		IO:     f.IO,
		Config: f.Config,
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

			return statusRun(opts)
		},
	}

	cmd.Flags().StringVarP(&opts.Hostname, "hostname", "h", "", "Check a specific instance's authentication status.")
	cmd.Flags().BoolVarP(&opts.ShowToken, "show-token", "t", false, "Display the authentication token.")

	return cmd
}

func statusRun(opts *StatusOpts) error {
	c := opts.IO.Color()
	cfg, err := opts.Config()
	if err != nil {
		return err
	}

	stderr := opts.IO.StdErr

	statusInfo := map[string][]string{}

	instances, err := cfg.Hosts()
	if len(instances) == 0 || err != nil {
		return fmt.Errorf("No GitLab instances have been authenticated with glab. Run `%s` to authenticate.\n", c.Bold("glab auth login"))
	}

	if opts.Hostname != "" && !slices.Contains(instances, opts.Hostname) {
		return fmt.Errorf("%s %s has not been authenticated with glab. Run `%s %s` to authenticate.", c.FailedIcon(), opts.Hostname, c.Bold("glab auth login --hostname"), c.Bold(opts.Hostname))
	}

	failedAuth := false
	for _, instance := range instances {
		if opts.Hostname != "" && opts.Hostname != instance {
			continue
		}
		statusInfo[instance] = []string{}
		addMsg := func(x string, ys ...interface{}) {
			statusInfo[instance] = append(statusInfo[instance], fmt.Sprintf(x, ys...))
		}

		token, tokenSource, _ := cfg.GetWithSource(instance, "token", false)
		apiClient, err := api.NewClientWithCfg(instance, cfg, false)
		if opts.HttpClientOverride != nil {
			apiClient, _ = opts.HttpClientOverride(token, instance)
		}
		if err == nil {
			user, err := api.CurrentUser(apiClient.Lab())
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
		apiEndpoint := glinstance.APIEndpoint(instance, apiProto)
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
			if opts.ShowToken {
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
		if opts.Hostname != "" && opts.Hostname != instance {
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
