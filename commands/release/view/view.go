package view

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/MakeNowJust/heredoc/v2"
	"github.com/spf13/cobra"
	"github.com/xanzy/go-gitlab"
	"gitlab.com/gitlab-org/cli/commands/cmdutils"
	"gitlab.com/gitlab-org/cli/commands/release/releaseutils"
	"gitlab.com/gitlab-org/cli/internal/config"
	"gitlab.com/gitlab-org/cli/internal/glrepo"
	"gitlab.com/gitlab-org/cli/pkg/iostreams"
	"gitlab.com/gitlab-org/cli/pkg/utils"
)

type ViewOpts struct {
	TagName       string
	OpenInBrowser bool

	IO         *iostreams.IOStreams
	HTTPClient func() (*gitlab.Client, error)
	BaseRepo   func() (glrepo.Interface, error)
	Config     func() (config.Config, error)
}

func NewCmdView(f *cmdutils.Factory) *cobra.Command {
	opts := &ViewOpts{
		IO:     f.IO,
		Config: f.Config,
	}

	cmd := &cobra.Command{
		Use:   "view <tag>",
		Short: "View information about a GitLab release.",
		Long: heredoc.Doc(`View information about a GitLab release.

			Without an explicit tag name argument, shows the latest release in the project.
		`),
		Example: heredoc.Doc(`
			# View the latest release of a GitLab repository
			$ glab release view

			# View a release with specified tag name
			$ glab release view v1.0.1
`),
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.HTTPClient = f.HttpClient
			opts.BaseRepo = f.BaseRepo

			if len(args) == 1 {
				opts.TagName = args[0]
			}

			return viewRun(opts)
		},
	}

	cmd.Flags().BoolVarP(&opts.OpenInBrowser, "web", "w", false, "Open the release in the browser.")

	return cmd
}

func viewRun(opts *ViewOpts) error {
	client, err := opts.HTTPClient()
	if err != nil {
		return err
	}

	repo, err := opts.BaseRepo()
	if err != nil {
		return err
	}

	cfg, _ := opts.Config()

	var resp *gitlab.Response
	var release *gitlab.Release

	if opts.TagName == "" {
		releases, _, err := client.Releases.ListReleases(repo.FullName(), &gitlab.ListReleasesOptions{})
		if err != nil {
			return cmdutils.WrapError(err, "could not fetch latest release.")
		}
		if len(releases) < 1 {
			return cmdutils.WrapError(errors.New("not found"), fmt.Sprintf("no release found for %q", repo.FullName()))
		}

		release = releases[0]
		opts.TagName = release.TagName
	} else {
		release, resp, err = client.Releases.GetRelease(repo.FullName(), opts.TagName)
		if err != nil {
			if resp != nil && (resp.StatusCode == http.StatusNotFound || resp.StatusCode == http.StatusForbidden) {
				return cmdutils.WrapError(err, "release does not exist.")
			}
			return cmdutils.WrapError(err, "failed to fetch release.")
		}
	}

	if opts.OpenInBrowser { // open in browser if --web flag is specified
		url := release.Links.Self

		if opts.IO.IsOutputTTY() {
			opts.IO.Logf("Opening %s in your browser.\n", url)
		}

		browser, _ := cfg.Get(repo.RepoHost(), "browser")
		return utils.OpenInBrowser(url, browser)
	}

	glamourStyle, _ := cfg.Get(repo.RepoHost(), "glamour_style")
	opts.IO.ResolveBackgroundColor(glamourStyle)

	err = opts.IO.StartPager()
	if err != nil {
		return err
	}
	defer opts.IO.StopPager()

	opts.IO.LogInfo(releaseutils.DisplayRelease(opts.IO, release, repo))
	return nil
}
