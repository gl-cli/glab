package update

import (
	"fmt"
	"os"
	"strings"
	"time"

	"gitlab.com/gitlab-org/cli/internal/config"
	"gitlab.com/gitlab-org/cli/internal/glrepo"
	"gitlab.com/gitlab-org/cli/internal/utils"

	"gitlab.com/gitlab-org/cli/internal/cmdutils"

	"github.com/MakeNowJust/heredoc/v2"
	"github.com/hashicorp/go-version"
	"github.com/spf13/cobra"
	gitlab "gitlab.com/gitlab-org/api/client-go"
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
		Long: heredoc.Doc(`Checks for new versions every 24 hours after any 'glab' command is run. Does not recheck if the most recent recheck is less than 24 hours old.

		To override the recheck behavior and force an update check, set the GLAB_CHECK_UPDATE environment variable to 'true'.

		To disable the update check entirely, run 'glab config set check_update false'.
		To re-enable the update check, run 'glab config set check_update true'.
		`),
		Aliases: commandAliases,
		RunE: func(cmd *cobra.Command, args []string) error {
			return CheckUpdate(f, false)
		},
	}

	return cmd
}

func CheckUpdate(f cmdutils.Factory, silentSuccess bool) error {
	moreThan24hAgo, err := checkLastUpdate(f)
	if err != nil {
		return err
	}
	// if the last update check was less than 24h ago we skip the version check
	if !moreThan24hAgo {
		return nil
	}

	// We set the project to the `glab` project to check for `glab` updates
	repo, err := glrepo.FromFullName(defaultProjectURL, f.DefaultHostname())
	if err != nil {
		return err
	}
	apiClient, err := f.ApiClient(repo.RepoHost(), f.Config())
	if err != nil {
		return err
	}
	gitlabClient := apiClient.Lab()

	// Since the `gitlab.com/gitlab-org/cli` is public, we remove the token
	// for this single request. When users have a `GITLAB_TOKEN` set with a
	// token for GitLab Self-Managed or GitLab Dedicated, we shouldn't use it
	// to authenticate to gitlab.com.
	releases, _, err := gitlabClient.Releases.ListReleases(
		repo.FullName(), &gitlab.ListReleasesOptions{ListOptions: gitlab.ListOptions{Page: 1, PerPage: 1}}, gitlab.WithToken(gitlab.PrivateToken, ""))
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

// Don't CheckUpdate if previous command is CheckUpdate
// or it’s Completion, so it doesn’t take a noticably long time
// to start new shells and we don’t encourage users setting
// `check_update` to false in the config.
func ShouldSkipUpdate(previousCommand string) bool {
	isCheckUpdate := previousCommand == commandUse || utils.PresentInStringSlice(commandAliases, previousCommand)
	isCompletion := previousCommand == "completion"

	return isCheckUpdate || isCompletion
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
// We only want to check for updates once every 24 hours
func checkLastUpdate(f cmdutils.Factory) (bool, error) {
	const updateCheckInterval = 24 * time.Hour
	cfg := f.Config()

	// We don't care when the command was run if the environment variable is forcing an update
	if isEnvForcingUpdate() {
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
