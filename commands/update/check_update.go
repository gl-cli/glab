package update

import (
	"fmt"
	"strings"

	"gitlab.com/gitlab-org/cli/pkg/utils"

	"gitlab.com/gitlab-org/cli/commands/cmdutils"

	"github.com/hashicorp/go-version"
	"github.com/spf13/cobra"
	"github.com/xanzy/go-gitlab"
)

const (
	defaultProjectURL = "https://gitlab.com/gitlab-org/cli"
	commandUse        = "check-update"
)

var commandAliases = []string{"update"}

func NewCheckUpdateCmd(f *cmdutils.Factory, version string) *cobra.Command {
	cmd := &cobra.Command{
		Use:     commandUse,
		Short:   "Check for latest glab releases.",
		Long:    ``,
		Aliases: commandAliases,
		RunE: func(cmd *cobra.Command, args []string) error {
			return CheckUpdate(f, version, false, "")
		},
	}

	return cmd
}

func CheckUpdate(f *cmdutils.Factory, version string, silentSuccess bool, previousCommand string) error {
	if shouldSkipUpdate(previousCommand) {
		return nil
	}

	// We set the project to the `glab` project to check for `glab` updates
	err := f.RepoOverride(defaultProjectURL)
	if err != nil {
		return err
	}
	repo, err := f.BaseRepo()
	if err != nil {
		return err
	}
	apiClient, err := f.HttpClient()
	if err != nil {
		return err
	}

	// Since the `gitlab.com/gitlab-org/cli` is public, we remove the token for this single request, so when users have
	// a `GITLAB_TOKEN` set with a token for their self-managed instance, we don't use it to authenticate to gitlab.com
	releases, _, err := apiClient.Releases.ListReleases(
		repo.FullName(), &gitlab.ListReleasesOptions{ListOptions: gitlab.ListOptions{Page: 1, PerPage: 1}}, gitlab.WithToken(gitlab.PrivateToken, ""))
	if err != nil {
		return fmt.Errorf("failed checking for glab updates: %s", err.Error())
	}
	if len(releases) < 1 {
		return fmt.Errorf("no release found for glab.")
	}
	latestRelease := releases[0]
	releaseURL := fmt.Sprintf("%s/-/releases/%s", defaultProjectURL, latestRelease.TagName)

	c := f.IO.Color()
	if isOlderVersion(latestRelease.Name, version) {
		fmt.Fprintf(f.IO.StdErr, "%s %s -> %s\n%s\n",
			c.Yellow("A new version of glab has been released:"),
			c.Red(version), c.Green(latestRelease.TagName),
			releaseURL)
	} else {
		if silentSuccess {
			return nil
		}
		fmt.Fprintf(f.IO.StdErr, "%v",
			c.Green("You are already using the latest version of glab!\n"))
	}
	return nil
}

// Don't CheckUpdate if previous command is CheckUpdate
// or it’s Completion, so it doesn’t take a noticably long time
// to start new shells and we don’t encourage users setting
// `check_update` to false in the config.
func shouldSkipUpdate(previousCommand string) bool {
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
