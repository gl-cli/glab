package login

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"regexp"
	"strings"

	"gitlab.com/gitlab-org/cli/internal/commands/auth/authutils"

	"gitlab.com/gitlab-org/cli/internal/iostreams"

	"github.com/AlecAivazis/survey/v2"
	"github.com/MakeNowJust/heredoc/v2"
	"github.com/spf13/cobra"
	"github.com/zalando/go-keyring"

	"gitlab.com/gitlab-org/cli/internal/api"
	"gitlab.com/gitlab-org/cli/internal/cmdutils"
	"gitlab.com/gitlab-org/cli/internal/config"
	"gitlab.com/gitlab-org/cli/internal/glinstance"
	"gitlab.com/gitlab-org/cli/internal/oauth2"
)

type LoginOptions struct {
	IO              *iostreams.IOStreams
	Config          func() config.Config
	apiClient       func(repoHost string, cfg config.Config) (*api.Client, error)
	defaultHostname string

	Interactive bool

	Hostname string
	Token    string
	JobToken string

	ApiHost     string
	ApiProtocol string
	GitProtocol string

	UseKeyring bool
}

var opts *LoginOptions

func NewCmdLogin(f cmdutils.Factory) *cobra.Command {
	opts = &LoginOptions{
		IO:              f.IO(),
		Config:          f.Config,
		apiClient:       f.ApiClient,
		defaultHostname: f.DefaultHostname(),
	}

	var tokenStdin bool

	cmd := &cobra.Command{
		Use:   "login",
		Args:  cobra.ExactArgs(0),
		Short: "Authenticate with a GitLab instance.",
		Long: heredoc.Docf(`
			Authenticate with a GitLab instance.
			You can pass in a token on standard input by using %[1]s--stdin%[1]s.
			The minimum required scopes for the token are: %[1]sapi%[1]s, %[1]swrite_repository%[1]s.
			Configuration and credentials are stored in the global configuration file (default %[1]s~/.config/glab-cli/config.yml%[1]s)
		`, "`"),
		Example: heredoc.Docf(`
			# Start interactive setup
			$ glab auth login

			# Authenticate against %[1]sgitlab.com%[1]s by reading the token from a file
			$ glab auth login --stdin < myaccesstoken.txt

			# Authenticate with GitLab Self-Managed or GitLab Dedicated
			$ glab auth login --hostname salsa.debian.org

			# Non-interactive setup
			$ glab auth login --hostname gitlab.example.org --token glpat-xxx --api-host gitlab.example.org:3443 --api-protocol https --git-protocol ssh

			# Non-interactive setup reading token from a file
			$ glab auth login --hostname gitlab.example.org --api-host gitlab.example.org:3443 --api-protocol https --git-protocol ssh  --stdin < myaccesstoken.txt

			# Non-interactive CI/CD setup
			$ glab auth login --hostname $CI_SERVER_HOST --job-token $CI_JOB_TOKEN
		`, "`"),
		RunE: func(cmd *cobra.Command, args []string) error {
			if !opts.IO.PromptEnabled() && !tokenStdin && opts.Token == "" && opts.JobToken == "" {
				return &cmdutils.FlagError{Err: errors.New("'--stdin', '--token', or '--job-token' required when not running interactively.")}
			}

			if opts.JobToken != "" && (opts.Token != "" || tokenStdin) {
				return &cmdutils.FlagError{Err: errors.New("specify one of '--job-token' or '--token' or '--stdin'. You cannot use more than one of these at the same time.")}
			}

			if opts.Token != "" && tokenStdin {
				return &cmdutils.FlagError{Err: errors.New("specify one of '--token' or '--stdin'. You cannot use both flags at the same time.")}
			}

			if tokenStdin {
				defer opts.IO.In.Close()
				token, err := io.ReadAll(opts.IO.In)
				if err != nil {
					return fmt.Errorf("failed to read token from STDIN: %w", err)
				}
				opts.Token = strings.TrimSpace(string(token))
			}

			if opts.IO.PromptEnabled() && opts.Token == "" && opts.JobToken == "" && opts.IO.IsaTTY {
				opts.Interactive = true
			}

			if cmd.Flags().Changed("hostname") {
				if err := hostnameValidator(opts.Hostname); err != nil {
					return &cmdutils.FlagError{Err: fmt.Errorf("error parsing '--hostname': %w", err)}
				}
			}

			if !opts.Interactive && opts.Hostname == "" {
				opts.Hostname = glinstance.DefaultHostname
			}

			if opts.Interactive && (opts.ApiHost != "" || opts.ApiProtocol != "" || opts.GitProtocol != "") {
				return &cmdutils.FlagError{Err: errors.New("api-host, api-protocol, and git-protocol can only be used in non-interactive mode.")}
			}

			if err := loginRun(cmd.Context(), opts); err != nil {
				return cmdutils.WrapError(err, "Could not sign in!")
			}

			return nil
		},
	}

	cmd.Flags().StringVarP(&opts.Hostname, "hostname", "h", "", "The hostname of the GitLab instance to authenticate with.")
	cmd.Flags().StringVarP(&opts.Token, "token", "t", "", "Your GitLab access token.")
	cmd.Flags().StringVarP(&opts.JobToken, "job-token", "j", "", "CI job token.")
	cmd.Flags().BoolVar(&tokenStdin, "stdin", false, "Read token from standard input.")
	cmd.Flags().BoolVar(&opts.UseKeyring, "use-keyring", false, "Store token in your operating system's keyring.")
	cmd.Flags().StringVarP(&opts.ApiHost, "api-host", "a", "", "API host url.")
	cmd.Flags().StringVarP(&opts.ApiProtocol, "api-protocol", "p", "", "API protocol: https, http")
	cmd.Flags().StringVarP(&opts.GitProtocol, "git-protocol", "g", "", "Git protocol: ssh, https, http")

	return cmd
}

func loginRun(ctx context.Context, opts *LoginOptions) error {
	c := opts.IO.Color()
	cfg := opts.Config()

	if opts.Token != "" {
		if opts.Hostname == "" {
			return errors.New("empty hostname would leak `oauth_token`")
		}

		if opts.UseKeyring {
			return keyring.Set("glab:"+opts.Hostname, "", opts.Token)
		} else {
			err := cfg.Set(opts.Hostname, "token", opts.Token)
			if err != nil {
				return err
			}

			if token := config.GetFromEnv("token"); token != "" {
				fmt.Fprintf(opts.IO.StdErr, "%s One of %s environment variables is set. If you don't want to use it for glab, unset it.\n", c.Yellow("WARNING:"), strings.Join(config.EnvKeyEquivalence("token"), ", "))
			}
			if opts.ApiHost != "" {
				err = cfg.Set(opts.Hostname, "api_host", opts.ApiHost)
				if err != nil {
					return err
				}
			}

			if opts.ApiProtocol != "" {
				err = cfg.Set(opts.Hostname, "api_protocol", opts.ApiProtocol)
				if err != nil {
					return err
				}
			}

			if opts.GitProtocol != "" {
				err = cfg.Set(opts.Hostname, "git_protocol", opts.GitProtocol)
				if err != nil {
					return err
				}
			}

			return cfg.Write()
		}

	}

	if opts.JobToken != "" {
		if opts.Hostname == "" {
			return errors.New("empty hostname would leak `oauth_token`")
		}

		if opts.UseKeyring {
			return keyring.Set("glab:"+opts.Hostname, "", opts.JobToken)
		} else {
			err := cfg.Set(opts.Hostname, "job_token", opts.JobToken)
			if err != nil {
				return err
			}

			if opts.ApiHost != "" {
				err = cfg.Set(opts.Hostname, "api_host", opts.ApiHost)
				if err != nil {
					return err
				}
			}

			if opts.ApiProtocol != "" {
				err = cfg.Set(opts.Hostname, "api_protocol", opts.ApiProtocol)
				if err != nil {
					return err
				}
			}

			if opts.GitProtocol != "" {
				err = cfg.Set(opts.Hostname, "git_protocol", opts.GitProtocol)
				if err != nil {
					return err
				}
			}

			return cfg.Write()
		}
	}

	hostname := opts.Hostname
	apiHostname := opts.Hostname

	if opts.ApiHost != "" {
		apiHostname = opts.ApiHost
	}

	isSelfHosted := false

	if hostname == "" {
		var hostType int
		err := survey.AskOne(&survey.Select{
			Message: "What GitLab instance do you want to sign in to?",
			Options: []string{
				opts.defaultHostname,
				"GitLab Self-Managed or GitLab Dedicated instance",
			},
		}, &hostType)
		if err != nil {
			return fmt.Errorf("could not prompt: %w", err)
		}

		isSelfHosted = hostType == 1

		hostname = opts.defaultHostname
		apiHostname = hostname
		if isSelfHosted {
			err := survey.AskOne(&survey.Input{
				Message: "GitLab hostname:",
			}, &hostname, survey.WithValidator(hostnameValidator))
			if err != nil {
				return fmt.Errorf("could not prompt: %w", err)
			}
			err = survey.AskOne(&survey.Input{
				Message: "API hostname:",
				Help:    "For instances with a different hostname for the API endpoint.",
				Default: hostname,
			}, &apiHostname, survey.WithValidator(hostnameValidator))
			if err != nil {
				return fmt.Errorf("could not prompt: %w", err)
			}
		}
	} else {
		isSelfHosted = glinstance.IsSelfHosted(hostname)
	}

	fmt.Fprintf(opts.IO.StdErr, "- Signing into %s\n", hostname)

	if token := config.GetFromEnv("token"); token != "" {
		fmt.Fprintf(opts.IO.StdErr, "%s One of %s environment variables is set. If you don't want to use it for glab, unset it.\n", c.Yellow("WARNING:"), strings.Join(config.EnvKeyEquivalence("token"), ", "))
	}
	existingToken, _, _ := cfg.GetWithSource(hostname, "token", false)

	if existingToken != "" && opts.Interactive {
		apiClient, err := opts.apiClient(hostname, cfg)
		if err != nil {
			return err
		}

		user, _, err := apiClient.Lab().Users.CurrentUser()
		if err == nil {
			username := user.Username
			var keepGoing bool
			err = survey.AskOne(&survey.Confirm{
				Message: fmt.Sprintf(
					"You're already logged into %s as %s. Do you want to re-authenticate?",
					hostname,
					username),
				Default: false,
			}, &keepGoing)
			if err != nil {
				return fmt.Errorf("could not prompt: %w", err)
			}

			if !keepGoing {
				return nil
			}
		}
	}

	var (
		loginType                string
		containerRegistryDomains string
	)

	if opts.Interactive {
		err := survey.AskOne(&survey.Select{
			Message: "How would you like to sign in?",
			Options: []string{
				"Token",
				"Web",
			},
		}, &loginType)
		if err != nil {
			return fmt.Errorf("could not get sign-in type: %w", err)
		}

		err = survey.AskOne(&survey.Input{
			Message: "What domains does this host use for the container registry and image dependency proxy?",
			Default: defaultContainerRegistryDomainsString(hostname),
		}, &containerRegistryDomains)
		if err != nil {
			return fmt.Errorf("could not get container registry domains: %w", err)
		}
	}

	var token string
	var err error
	if strings.EqualFold(loginType, "token") {
		token, err = showTokenPrompt(opts.IO, hostname)
		if err != nil {
			return err
		}
	} else {
		client, err := opts.apiClient(hostname, cfg)
		if err != nil {
			return err
		}

		token, err = oauth2.StartFlow(ctx, cfg, opts.IO.StdErr, client.HTTPClient(), hostname)
		if err != nil {
			return err
		}
	}

	if opts.UseKeyring {
		err = keyring.Set("glab:"+hostname, "", token)
		if err != nil {
			return err
		}
	} else {
		err = cfg.Set(hostname, "token", token)
		if err != nil {
			return err
		}

		err = setContainerRegistryDomains(cfg, hostname, containerRegistryDomains)
		if err != nil {
			return err
		}
	}

	if hostname == "" {
		return errors.New("empty hostname would leak the token")
	}

	err = cfg.Set(hostname, "api_host", apiHostname)
	if err != nil {
		return err
	}

	gitProtocol := "https"
	apiProtocol := "https"

	glabExecutable := "glab"
	if exe, err := os.Executable(); err == nil {
		glabExecutable = exe
	}
	credentialFlow := &authutils.GitCredentialFlow{Executable: glabExecutable}

	if opts.Interactive {
		err = survey.AskOne(&survey.Select{
			Message: "Choose default Git protocol:",
			Options: []string{
				"SSH",
				"HTTPS",
				"HTTP",
			},
			Default: "HTTPS",
		}, &gitProtocol)
		if err != nil {
			return fmt.Errorf("could not prompt: %w", err)
		}

		gitProtocol = strings.ToLower(gitProtocol)
		if opts.Interactive && gitProtocol != "ssh" {
			if err := credentialFlow.Prompt(hostname, gitProtocol); err != nil {
				return err
			}
		}

		if isSelfHosted {
			err = survey.AskOne(&survey.Select{
				Message: "Choose host API protocol:",
				Options: []string{
					"HTTPS",
					"HTTP",
				},
				Default: "HTTPS",
			}, &apiProtocol)
			if err != nil {
				return fmt.Errorf("could not prompt: %w", err)
			}

			apiProtocol = strings.ToLower(apiProtocol)
		}

		fmt.Fprintf(opts.IO.StdErr, "- glab config set -h %s git_protocol %s\n", hostname, gitProtocol)
		err = cfg.Set(hostname, "git_protocol", gitProtocol)
		if err != nil {
			return err
		}

		fmt.Fprintf(opts.IO.StdErr, "%s Configured Git protocol.\n", c.GreenCheck())

		fmt.Fprintf(opts.IO.StdErr, "- glab config set -h %s api_protocol %s\n", hostname, apiProtocol)
		err = cfg.Set(hostname, "api_protocol", apiProtocol)
		if err != nil {
			return err
		}

		fmt.Fprintf(opts.IO.StdErr, "%s Configured API protocol.\n", c.GreenCheck())
	}
	apiClient, err := opts.apiClient(hostname, cfg)
	if err != nil {
		return err
	}

	user, _, err := apiClient.Lab().Users.CurrentUser()
	if err != nil {
		return fmt.Errorf("error using API: %w", err)
	}
	username := user.Username

	err = cfg.Set(hostname, "user", username)
	if err != nil {
		return err
	}

	err = cfg.Write()
	if err != nil {
		return err
	}

	if credentialFlow.ShouldSetup() {
		err := credentialFlow.Setup(hostname, gitProtocol, username, token)
		if err != nil {
			return err
		}
	}

	fmt.Fprintf(opts.IO.StdErr, "%s Logged in as %s\n", c.GreenCheck(), c.Bold(username))
	fmt.Fprintf(opts.IO.StdErr, "%s Configuration saved to %s\n", c.GreenCheck(), config.ConfigFile())

	return nil
}

func hostnameValidator(v any) error {
	val := fmt.Sprint(v)
	if len(strings.TrimSpace(val)) < 1 {
		return errors.New("a value is required.")
	}
	re := regexp.MustCompile(`^(([a-z0-9]|[a-z0-9][a-z0-9\-]*[a-z0-9])\.)*([a-z0-9]|[a-z0-9][a-z0-9\-]*[a-z0-9])(:[0-9]+)?(/[a-z0-9]*)*$`)
	if !re.MatchString(val) {
		return fmt.Errorf("invalid hostname %q", val)
	}
	return nil
}

func getAccessTokenTip(hostname string) string {
	return fmt.Sprintf(`
	Tip: generate a personal access token at https://%s/-/user_settings/personal_access_tokens?scopes=api,write_repository.
	The minimum required scopes are 'api' and 'write_repository'.`, hostname)
}

func showTokenPrompt(io *iostreams.IOStreams, hostname string) (string, error) {
	fmt.Fprintln(io.StdErr)
	fmt.Fprintln(io.StdErr, heredoc.Doc(getAccessTokenTip(hostname)))

	var token string
	err := survey.AskOne(&survey.Password{
		Message: "Paste your authentication token:",
	}, &token, survey.WithValidator(survey.Required))
	if err != nil {
		return "", fmt.Errorf("could not prompt: %w", err)
	}

	return token, nil
}

func defaultContainerRegistryDomainsString(hostname string) string {
	if !strings.Contains(hostname, ":") {
		return strings.Join(
			[]string{
				hostname,
				net.JoinHostPort(hostname, "443"),
				"registry." + hostname,
			}, ",")
	}

	return strings.Join(
		[]string{
			hostname,
			"registry." + hostname,
		}, ",")
}

func setContainerRegistryDomains(cfg config.Config, hostname string, domains string) error {
	return cfg.Set(hostname, "container_registry_domains", domains)
}
