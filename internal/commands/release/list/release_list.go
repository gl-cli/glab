package list

import (
	"fmt"

	"gitlab.com/gitlab-org/cli/internal/cmdutils"
	"gitlab.com/gitlab-org/cli/internal/commands/release/releaseutils"
	"gitlab.com/gitlab-org/cli/internal/utils"

	"github.com/spf13/cobra"
	gitlab "gitlab.com/gitlab-org/api/client-go"
)

var getRelease = func(client *gitlab.Client, projectID any, tag string) (*gitlab.Release, error) {
	release, _, err := client.Releases.GetRelease(projectID, tag)
	if err != nil {
		return nil, err
	}

	return release, nil
}

var listReleases = func(client *gitlab.Client, projectID any, opts *gitlab.ListReleasesOptions) ([]*gitlab.Release, error) {
	releases, _, err := client.Releases.ListReleases(projectID, opts)
	return releases, err
}

func NewCmdReleaseList(f cmdutils.Factory) *cobra.Command {
	releaseListCmd := &cobra.Command{
		Use:     "list [flags]",
		Short:   `List releases in a repository.`,
		Long:    ``,
		Aliases: []string{"ls"},
		Args:    cobra.ExactArgs(0),
		RunE: func(cmd *cobra.Command, args []string) error {
			return run(f, cmd)
		},
	}

	releaseListCmd.Flags().IntP("page", "p", 1, "Page number.")
	releaseListCmd.Flags().IntP("per-page", "P", 30, "Number of items to list per page.")

	releaseListCmd.Flags().StringP("tag", "t", "", "Filter releases by tag <name>.")
	// deprecate in favour of the `release view` command
	_ = releaseListCmd.Flags().MarkDeprecated("tag", "Use `glab release view <tag>` instead.")

	// make it hidden but still accessible
	// TODO: completely remove before a major release (v2.0.0+)
	_ = releaseListCmd.Flags().MarkHidden("tag")

	return releaseListCmd
}

func run(factory cmdutils.Factory, cmd *cobra.Command) error {
	l := &gitlab.ListReleasesOptions{}

	page, _ := cmd.Flags().GetInt("page")
	l.Page = page
	perPage, _ := cmd.Flags().GetInt("per-page")
	l.PerPage = perPage

	tag, err := cmd.Flags().GetString("tag")
	if err != nil {
		return err
	}

	apiClient, err := factory.HttpClient()
	if err != nil {
		return err
	}

	repo, err := factory.BaseRepo()
	if err != nil {
		return err
	}

	if tag != "" {
		release, err := getRelease(apiClient, repo.FullName(), tag)
		if err != nil {
			return err
		}

		cfg := factory.Config()
		glamourStyle, _ := cfg.Get(repo.RepoHost(), "glamour_style")
		factory.IO().ResolveBackgroundColor(glamourStyle)

		err = factory.IO().StartPager()
		if err != nil {
			return err
		}
		defer factory.IO().StopPager()

		fmt.Fprintln(factory.IO().StdOut, releaseutils.DisplayRelease(factory.IO(), release, repo))
	} else {

		releases, err := listReleases(apiClient, repo.FullName(), l)
		if err != nil {
			return err
		}

		title := utils.NewListTitle("release")
		title.RepoName = repo.FullName()
		title.Page = 0
		title.CurrentPageTotal = len(releases)
		err = factory.IO().StartPager()
		if err != nil {
			return err
		}
		defer factory.IO().StopPager()

		fmt.Fprintf(factory.IO().StdOut, "%s\n%s\n", title.Describe(), releaseutils.DisplayAllReleases(factory.IO(), releases, repo.FullName()))
	}
	return nil
}
