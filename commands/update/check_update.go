package update

import (
	"fmt"
	"strings"

	"gitlab.com/gitlab-org/cli/commands/cmdutils"

	"github.com/hashicorp/go-version"
	"github.com/spf13/cobra"
	"github.com/xanzy/go-gitlab"
)

const defaultProjectURL = "https://gitlab.com/gitlab-org/cli"

func NewCheckUpdateCmd(f *cmdutils.Factory, version string) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "check-update",
		Short:   "Check for latest glab releases",
		Long:    ``,
		Aliases: []string{"update"},
		RunE: func(cmd *cobra.Command, args []string) error {
			return CheckUpdate(f, version, false)
		},
	}

	return cmd
}

func CheckUpdate(f *cmdutils.Factory, version string, silentErr bool) error {
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
	releases, _, err := apiClient.Releases.ListReleases(repo.FullName(), &gitlab.ListReleasesOptions{Page: 1, PerPage: 1})
	if err != nil {
		if silentErr {
			return nil
		}
		return fmt.Errorf("could not check for update: %s", err.Error())
	}
	if len(releases) < 1 {
		return fmt.Errorf("no release found for glab")
	}
	latestRelease := releases[0]
	releaseURL := fmt.Sprintf("%s/-/releases/%s", defaultProjectURL, latestRelease.TagName)

	c := f.IO.Color()
	if isOlderVersion(latestRelease.Name, version) {
		fmt.Fprintf(f.IO.StdErr, "%s %s → %s\n%s\n",
			c.Yellow("A new version of glab has been released:"),
			c.Red(version), c.Green(latestRelease.TagName),
			releaseURL)
	} else {
		if silentErr {
			return nil
		}
		fmt.Fprintf(f.IO.StdErr, "%v %v", c.GreenCheck(),
			c.Green("You are already using the latest version of glab\n"))
	}
	return nil
}

func isOlderVersion(latestVersion, appVersion string) bool {
	latestVersion = strings.TrimSpace(latestVersion)
	appVersion = strings.TrimSpace(appVersion)

	vv, ve := version.NewVersion(latestVersion)
	vw, we := version.NewVersion(appVersion)

	return ve == nil && we == nil && vv.GreaterThan(vw)
}
