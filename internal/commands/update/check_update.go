package update

import (
	"errors"
	"fmt"
	"net"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/MakeNowJust/heredoc/v2"
	"github.com/hashicorp/go-version"
	"github.com/spf13/cobra"

	gitlab "gitlab.com/gitlab-org/api/client-go"

	"gitlab.com/gitlab-org/cli/internal/api"
	"gitlab.com/gitlab-org/cli/internal/cmdutils"
	"gitlab.com/gitlab-org/cli/internal/config"
	"gitlab.com/gitlab-org/cli/internal/glinstance"
	"gitlab.com/gitlab-org/cli/internal/iostreams"
	"gitlab.com/gitlab-org/cli/internal/mcpannotations"
	"gitlab.com/gitlab-org/cli/internal/utils"
)

const (
	defaultProjectURL = "https://gitlab.com/gitlab-org/cli"
	commandUse        = "check-update"
)

var commandAliases = []string{"update"}

func NewCheckUpdateCmd(f cmdutils.Factory) *cobra.Command {
	cmd := &cobra.Command{
		Use:   commandUse,
		Short: "Check for latest glab releases.",
		Long: heredoc.Doc(`Checks for the latest version of glab available on GitLab.com.

		When run explicitly, this command always checks for updates regardless of when the last check occurred.

		When run automatically after other glab commands, it checks for updates at most once every 24 hours.

		To disable the automatic update check entirely, run 'glab config set check_update false'.
		To re-enable the automatic update check, run 'glab config set check_update true'.
		`),
		Aliases: commandAliases,
		Annotations: map[string]string{
			mcpannotations.Safe: "true",
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			return CheckUpdateExplicit(f)
		},
	}

	return cmd
}

// clientCreator is a variable that can be overridden for testing
var clientCreator = createUnauthenticatedClient

func CheckUpdate(f cmdutils.Factory, silentSuccess bool) error {
	return checkUpdate(f, silentSuccess, false)
}

// CheckUpdateExplicit performs an update check when explicitly invoked by the user.
// Unlike automatic checks, this bypasses the 24-hour throttle.
func CheckUpdateExplicit(f cmdutils.Factory) error {
	return checkUpdate(f, false, true)
}

func checkUpdate(f cmdutils.Factory, silentSuccess bool, forceCheck bool) error {
	moreThan24hAgo, err := checkLastUpdate(f, forceCheck)
	if err != nil {
		return err
	}
	// if the last update check was less than 24h ago we skip the version check
	// (unless this is a forced check from explicit command invocation)
	if !moreThan24hAgo {
		return nil
	}

	// Create an unauthenticated API client to check for updates on the public gitlab.com/gitlab-org/cli project.
	// We explicitly avoid using user credentials since:
	// 1. The releases endpoint is public and doesn't require authentication
	// 2. Using user credentials (especially from GITLAB_TOKEN env var) can cause issues
	//    when users have tokens for self-hosted instances that aren't valid for gitlab.com
	apiClient, err := clientCreator(f.BuildInfo().UserAgent())
	if err != nil {
		return err
	}
	gitlabClient := apiClient.Lab()

	releases, _, err := gitlabClient.Releases.ListReleases(
		"gitlab-org/cli", &gitlab.ListReleasesOptions{ListOptions: gitlab.ListOptions{Page: 1, PerPage: 1}})
	if err != nil {
		return fmt.Errorf("failed checking for glab updates: %s", err.Error())
	}
	if len(releases) < 1 {
		return fmt.Errorf("no release found for glab.")
	}
	latestRelease := releases[0]
	releaseURL := fmt.Sprintf("%s/-/releases/%s", defaultProjectURL, latestRelease.TagName)

	version := f.BuildInfo().Version

	c := f.IO().Color()
	if isOlderVersion(latestRelease.Name, version) {
		fmt.Fprintf(f.IO().StdErr, "%s %s -> %s\n%s\n",
			c.Yellow("A new version of glab has been released:"),
			c.Red(version), c.Green(latestRelease.TagName),
			releaseURL)
	} else {
		if silentSuccess {
			return nil
		}
		fmt.Fprintf(f.IO().StdErr, "%v",
			c.Green("You are already using the latest version of glab!\n"))
	}
	return nil
}

// createUnauthenticatedClient creates an API client without authentication for accessing
// public endpoints on gitlab.com. This avoids issues where user credentials (especially
// from environment variables like GITLAB_TOKEN) might be for self-hosted instances
// and invalid for gitlab.com.
func createUnauthenticatedClient(userAgent string, options ...api.ClientOption) (*api.Client, error) {
	// Create a client with an empty token for unauthenticated requests
	opts := []api.ClientOption{
		api.WithBaseURL(glinstance.APIEndpoint(glinstance.DefaultHostname, glinstance.DefaultProtocol, "")),
		api.WithUserAgent(userAgent),
	}
	opts = append(opts, options...)

	return api.NewClient(
		func(c *http.Client) (gitlab.AuthSource, error) {
			// Use AccessTokenAuthSource with empty token for public API access
			return gitlab.AccessTokenAuthSource{Token: ""}, nil
		},
		opts...,
	)
}

// Don't CheckUpdate if previous command is CheckUpdate
// or it's Completion, so it doesn't take a noticably long time
// to start new shells and we don't encourage users setting
// `check_update` to false in the config.
// Also skip for git-credential to avoid interfering with Git operations.
func ShouldSkipUpdate(previousCommand string) bool {
	isCheckUpdate := previousCommand == commandUse || utils.PresentInStringSlice(commandAliases, previousCommand)
	isCompletion := previousCommand == "completion"
	isGitCredential := previousCommand == "git-credential"

	return isCheckUpdate || isCompletion || isGitCredential
}

func isOlderVersion(latestVersion, appVersion string) bool {
	latestVersion = strings.TrimSpace(latestVersion)
	appVersion = strings.TrimSpace(appVersion)

	vv, ve := version.NewVersion(latestVersion)
	vw, we := version.NewVersion(appVersion)

	return ve == nil && we == nil && vv.GreaterThan(vw)
}

// returns true if we should check for updates
//
// returns false if we should skip the update check
//
// We only want to check for updates once every 24 hours, unless forceCheck is true
func checkLastUpdate(f cmdutils.Factory, forceCheck bool) (bool, error) {
	const updateCheckInterval = 24 * time.Hour
	cfg := f.Config()

	// We don't care when the command was run if the environment variable is forcing an update
	// or if this is an explicit command invocation (forceCheck = true)
	if isEnvForcingUpdate() || forceCheck {
		if err := updateLastCheckTimestamp(cfg); err != nil {
			return false, err
		}
		return true, nil
	}

	last_update, err := cfg.Get("", "last_update_check_timestamp")
	if err != nil {
		return false, err
	}

	// this might be the first time running the command, so last_update might be empty
	// we want to save the current time and check for an update
	if last_update == "" {
		if err := updateLastCheckTimestamp(cfg); err != nil {
			return false, err
		}
		return true, nil
	}

	last_update_time, err := time.Parse(time.RFC3339, last_update)
	if err != nil {
		return false, err
	}

	// if the last check was more than 24h ago we check for an update
	moreThan24hAgo := time.Since(last_update_time) > updateCheckInterval
	if moreThan24hAgo {
		if err := updateLastCheckTimestamp(cfg); err != nil {
			return false, err
		}
	}
	return moreThan24hAgo, nil
}

// isEnvForcingUpdate - returns true if the environment variable `GLAB_CHECK_UPDATE` is set to true
func isEnvForcingUpdate() bool {
	if envVal, ok := os.LookupEnv("GLAB_CHECK_UPDATE"); ok {
		switch strings.ToUpper(envVal) {
		case "TRUE", "YES", "Y", "1":
			return true
		case "FALSE", "NO", "N", "0":
			return false
		}
	}
	// if the value is not set or is not a valid value
	return false
}

// updateLastCheckTimestamp - saves the current time as last_update_check_timestamp to config.yml
func updateLastCheckTimestamp(cfg config.Config) error {
	if err := cfg.Set("", "last_update_check_timestamp", time.Now().Format(time.RFC3339)); err != nil {
		return err
	}

	if err := cfg.Write(); err != nil {
		return err
	}

	return nil
}

// PrintUpdateError prints update check errors with helpful formatting and context.
// This is specifically for errors that occur during background update checks,
// not main command execution errors.
func PrintUpdateError(streams *iostreams.IOStreams, err error, cmd *cobra.Command, debug bool) {
	color := streams.Color()

	var dnsError *net.DNSError
	if errors.As(err, &dnsError) {
		streams.LogErrorf("%s error connecting to %s\n", color.FailedIcon(), dnsError.Name)
		if debug {
			streams.LogError(color.FailedIcon(), dnsError)
		}
		streams.LogInfof("%s Check your internet connection and status.gitlab.com. If on GitLab Self-Managed, run 'sudo gitlab-ctl status' on your server.\n", color.DotWarnIcon())
	} else {
		var exitError *cmdutils.ExitError
		if errors.As(err, &exitError) {
			streams.LogErrorf("%s %s %s=%s\n", color.FailedIcon(), color.Bold(exitError.Details), color.Red("error"), exitError.Err)
		} else {
			streams.LogError("ERROR:", err)

			var flagError *cmdutils.FlagError
			if errors.As(err, &flagError) || strings.HasPrefix(err.Error(), "unknown command ") {
				if cmd != nil {
					streams.LogInfof("Try '%s --help' for more information.", cmd.CommandPath())
				} else {
					streams.LogInfof("Try --help for more information.")
				}
			}
		}
	}

	if cmd != nil {
		cmd.Print("\n")
	}
}
