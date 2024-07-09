package generate

import (
	"errors"
	"fmt"
	"time"

	"github.com/MakeNowJust/heredoc/v2"
	"gitlab.com/gitlab-org/cli/api"

	"github.com/spf13/cobra"
	"github.com/xanzy/go-gitlab"
	"gitlab.com/gitlab-org/cli/commands/cmdutils"
	"gitlab.com/gitlab-org/cli/pkg/git"
)

func NewCmdGenerate(f *cmdutils.Factory) *cobra.Command {
	changelogGenerateCmd := &cobra.Command{
		Use:   "generate [flags]",
		Short: `Generate a changelog for the repository or project.`,
		Long:  ``,
		Example: heredoc.Doc(`
			glab changelog generate
		`),
		Args: cobra.ExactArgs(0),
		RunE: func(cmd *cobra.Command, args []string) error {
			apiClient, err := f.HttpClient()
			if err != nil {
				return err
			}

			repo, err := f.BaseRepo()
			if err != nil {
				return err
			}

			opts := &gitlab.GenerateChangelogDataOptions{}

			// Set the version
			if s, _ := cmd.Flags().GetString("version"); s != "" {
				opts.Version = gitlab.Ptr(s)
			} else {
				tags, err := git.ListTags()
				if err != nil {
					return err
				}

				if len(tags) == 0 {
					return errors.New("no tags found. Either fetch tags, or pass a version with --version instead.")
				}

				version, err := git.DescribeByTags()
				if err != nil {
					return fmt.Errorf("failed to determine version from `git describe`: %w..", err)
				}
				opts.Version = gitlab.Ptr(version)
			}

			// Set the config file
			if s, _ := cmd.Flags().GetString("config-file"); s != "" {
				opts.ConfigFile = gitlab.Ptr(s)
			}

			// Set the date
			if s, _ := cmd.Flags().GetString("date"); s != "" {
				parsedDate, err := time.Parse(time.RFC3339, s)
				if err != nil {
					return err
				}

				t := gitlab.ISOTime(parsedDate)
				opts.Date = &t
			}

			// Set the "from" attribute
			if s, _ := cmd.Flags().GetString("from"); s != "" {
				opts.From = gitlab.Ptr(s)
			}

			// Set the "to" attribute
			if s, _ := cmd.Flags().GetString("to"); s != "" {
				opts.To = gitlab.Ptr(s)
			}

			// Set the trailer
			if s, _ := cmd.Flags().GetString("trailer"); s != "" {
				opts.Trailer = gitlab.Ptr(s)
			}

			project, err := repo.Project(apiClient)
			if err != nil {
				return err
			}

			changelog, err := api.GenerateChangelog(apiClient, project.ID, opts)
			if err != nil {
				return err
			}

			fmt.Fprintf(f.IO.StdOut, "%s", changelog.Notes)

			return nil
		},
	}

	// The options mimic the ones from the REST API.
	// https://docs.gitlab.com/ee/api/repositories.html#generate-changelog-data
	changelogGenerateCmd.Flags().StringP("version", "v", "", "Version to generate the changelog for. Must follow semantic versioning. Defaults to the version of the local checkout, like using 'git describe'.")
	changelogGenerateCmd.Flags().StringP("config-file", "", "", "Path of the changelog configuration file in the project's Git repository. Defaults to '.gitlab/changelog_config.yml'.")
	changelogGenerateCmd.Flags().StringP("date", "", "", "Date and time of the release. Uses ISO 8601 (`2016-03-11T03:45:40Z`) format. Defaults to the current time.")
	changelogGenerateCmd.Flags().StringP("from", "", "", "Start of the range of commits (as a SHA) to use when generating the changelog. This commit itself isn't included in the list.")
	changelogGenerateCmd.Flags().StringP("to", "", "", "End of the range of commits (as a SHA) to use when generating the changelog. This commit is included in the list. Defaults to the HEAD of the project's default branch.")
	changelogGenerateCmd.Flags().StringP("trailer", "", "", "The Git trailer to use for including commits. Defaults to 'Changelog'.")

	return changelogGenerateCmd
}
