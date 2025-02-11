package create

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	securejoin "github.com/cyphar/filepath-securejoin"
	catalog "gitlab.com/gitlab-org/cli/commands/release/create/catalog"
	"gitlab.com/gitlab-org/cli/commands/release/releaseutils"
	"gitlab.com/gitlab-org/cli/commands/release/releaseutils/upload"

	"github.com/AlecAivazis/survey/v2"
	"github.com/MakeNowJust/heredoc/v2"
	"gitlab.com/gitlab-org/cli/internal/config"
	"gitlab.com/gitlab-org/cli/internal/run"
	"gitlab.com/gitlab-org/cli/pkg/git"
	"gitlab.com/gitlab-org/cli/pkg/prompt"
	"gitlab.com/gitlab-org/cli/pkg/surveyext"
	"gitlab.com/gitlab-org/cli/pkg/utils"

	"gitlab.com/gitlab-org/cli/internal/glrepo"
	"gitlab.com/gitlab-org/cli/pkg/iostreams"

	"github.com/spf13/cobra"
	gitlab "gitlab.com/gitlab-org/api/client-go"
	"gitlab.com/gitlab-org/cli/commands/cmdutils"
)

type CreateOpts struct {
	Name                        string
	Ref                         string
	TagName                     string
	TagMessage                  string
	Notes                       string
	NotesFile                   string
	ExperimentalNotesTextOrFile string
	Milestone                   []string
	AssetLinksAsJson            string
	ReleasedAt                  string
	RepoOverride                string
	PublishToCatalog            bool
	NoUpdate                    bool
	NoCloseMilestone            bool

	NoteProvided       bool
	ReleaseNotesAction string

	AssetLinks []*upload.ReleaseAsset
	AssetFiles []*upload.ReleaseFile

	IO         *iostreams.IOStreams
	HTTPClient func() (*gitlab.Client, error)
	BaseRepo   func() (glrepo.Interface, error)
	Config     func() (config.Config, error)
}

func NewCmdCreate(f *cmdutils.Factory) *cobra.Command {
	opts := &CreateOpts{
		IO:     f.IO,
		Config: f.Config,
	}

	cmd := &cobra.Command{
		Use:   "create <tag> [<files>...]",
		Short: "Create a new GitLab release, or update an existing one.",
		Long: heredoc.Docf(`Create a new release, or update an existing GitLab release, for a repository. Requires the Developer role or higher.

		An existing release is updated with the new information you provide.

		To create a release from an annotated Git tag, first create one locally with
		Git, push the tag to GitLab, then run this command.

		If the Git tag you specify doesn't exist, the release is created
		from the latest state of the default branch, and tagged with the tag name you specify.

		To override this behavior, use %[1]s--ref%[1]s. The %[1]sref%[1]s can be a commit SHA, another tag name, or a branch name.

		To fetch the new tag locally after the release, run %[1]sgit fetch --tags origin%[1]s.
		`, "`"),
		Args: cmdutils.MinimumArgs(1, "no tag name provided."),
		Example: heredoc.Docf(`
			# Create a release interactively
			$ glab release create v1.0.1

			# Create a release non-interactively by specifying a note
			$ glab release create v1.0.1 --notes "bugfix release"

			# Use release notes from a file
			$ glab release create v1.0.1 -F changelog.md

			# Upload a release asset with a display name (type will default to 'other')
			$ glab release create v1.0.1 '/path/to/asset.zip#My display label'

			# Upload a release asset with a display name and type
			$ glab release create v1.0.1 '/path/to/asset.png#My display label#image'

			# Upload all assets in a specified folder (types default to 'other')
			$ glab release create v1.0.1 ./dist/*

			# Upload all tarballs in a specified folder (types default to 'other')
			$ glab release create v1.0.1 ./dist/*.tar.gz

			# Create a release with assets specified as JSON object
			$ glab release create v1.0.1 --assets-links='
			  [
			    {
			      "name": "Asset1",
			      "url":"https://<domain>/some/location/1",
			      "link_type": "other",
			      "direct_asset_path": "path/to/file"
			    }
			  ]'

			# [EXPERIMENTAL] Create a release and publish it to the GitLab CI/CD catalog
			# Requires the feature flag %[1]sci_release_cli_catalog_publish_option%[1]s to be enabled
			# for this project in your GitLab instance. Do NOT run this manually. Use it as part
			# of a CI/CD pipeline with the "release" keyword:
			#
			# - The API endpoint accepts only "CI_JOB_TOKEN" as the authentication token.
			# - It retrieves components from the current repository by searching for
			#   %[1]syml%[1]s files within the "templates" directory and its subdirectories.
			# - It fails if the feature flag %[1]sci_release_cli_catalog_publish_option%[1]s
			#   is not enabled for this project in your GitLab instance.

			# Components can be defined:

			# - In single files ending in %[1]s.yml%[1]s for each component, like %[1]stemplates/secret-detection.yml%[1]s.
			# - In subdirectories containing %[1]stemplate.yml%[1]s files as entry points,
			#   for components that bundle together multiple related files. For example,
			#   %[1]stemplates/secret-detection/template.yml%[1]s.

			$ glab release create v1.0.1 --publish-to-catalog
`, "`"),
		RunE: func(cmd *cobra.Command, args []string) error {
			var err error
			opts.RepoOverride, _ = cmd.Flags().GetString("repo")
			opts.HTTPClient = f.HttpClient
			opts.BaseRepo = f.BaseRepo

			opts.TagName = args[0]

			opts.AssetFiles, err = releaseutils.AssetsFromArgs(args[1:])
			if err != nil {
				return err
			}

			if opts.AssetLinksAsJson != "" {
				err := json.Unmarshal([]byte(opts.AssetLinksAsJson), &opts.AssetLinks)
				if err != nil {
					return fmt.Errorf("failed to parse JSON string: %w", err)
				}
			}

			opts.Notes, err = resolveNotes(cmd, opts)
			if err != nil {
				return err
			}
			opts.NoteProvided = opts.Notes != ""

			return createRun(opts)
		},
	}

	cmd.Flags().StringVarP(&opts.Name, "name", "n", "", "The release name or title.")
	cmd.Flags().StringVarP(&opts.Ref, "ref", "r", "", "If the specified tag doesn't exist, the release is created from ref and tagged with the specified tag name. It can be a commit SHA, another tag name, or a branch name.")
	cmd.Flags().StringVarP(&opts.TagMessage, "tag-message", "T", "", "Message to use if creating a new annotated tag.")
	cmd.Flags().StringVarP(&opts.Notes, "notes", "N", "", "The release notes or description. You can use Markdown.")
	cmd.Flags().StringVarP(&opts.NotesFile, "notes-file", "F", "", "Read release notes 'file'. Specify '-' as the value to read from stdin.")
	cmd.Flags().StringVarP(&opts.ReleasedAt, "released-at", "D", "", "The 'date' when the release was ready. Defaults to the current datetime. Expects ISO 8601 format (2019-03-15T08:00:00Z).")
	cmd.Flags().StringSliceVarP(&opts.Milestone, "milestone", "m", []string{}, "The title of each milestone the release is associated with.")
	cmd.Flags().StringVarP(&opts.AssetLinksAsJson, "assets-links", "a", "", "'JSON' string representation of assets links, like `--assets-links='[{\"name\": \"Asset1\", \"url\":\"https://<domain>/some/location/1\", \"link_type\": \"other\", \"direct_asset_path\": \"path/to/file\"}]'.`")
	cmd.Flags().BoolVar(&opts.PublishToCatalog, "publish-to-catalog", false, "[EXPERIMENTAL] Publish the release to the GitLab CI/CD catalog.")
	cmd.Flags().BoolVar(&opts.NoUpdate, "no-update", false, "Prevent updating the existing release.")
	cmd.Flags().BoolVar(&opts.NoCloseMilestone, "no-close-milestone", false, "Prevent closing milestones after creating the release.")
	cmd.Flags().StringVar(&opts.ExperimentalNotesTextOrFile, "experimental-notes-text-or-file", "", "[EXPERIMENTAL] Value to use as release notes. If a file exists with this value as path, its content will be used. Otherwise, the value itself will be used as text.")
	_ = cmd.Flags().MarkHidden("experimental-notes-text-or-file")

	// These two need to be separately exclusive to avoid a breaking change
	// because there may be existing scripts that already use both notes and notes-file.
	cmd.MarkFlagsMutuallyExclusive("experimental-notes-text-or-file", "notes")
	cmd.MarkFlagsMutuallyExclusive("experimental-notes-text-or-file", "notes-file")

	return cmd
}

func resolveNotes(cmd *cobra.Command, opts *CreateOpts) (string, error) {
	if cmd.Flags().Changed("notes") {
		return opts.Notes, nil
	}

	if opts.NotesFile != "" {
		return resolveNotesFile(opts)
	}

	if opts.ExperimentalNotesTextOrFile != "" {
		return resolveNotesFileOrText(opts)
	}

	return "", nil
}

func resolveNotesFile(opts *CreateOpts) (string, error) {
	var b []byte
	var err error

	if opts.NotesFile == "-" {
		b, err = io.ReadAll(opts.IO.In)
		_ = opts.IO.In.Close()
	} else {
		b, err = os.ReadFile(opts.NotesFile)
	}

	if err != nil {
		return "", err
	}

	return string(b), nil
}

func resolveNotesFileOrText(opts *CreateOpts) (string, error) {
	// Rules from: https://docs.gitlab.com/ee/ci/yaml/index.html#releasedescription

	// Rule 1: A file path can't have spaces
	if strings.Contains(strings.TrimSpace(opts.ExperimentalNotesTextOrFile), " ") {
		return opts.ExperimentalNotesTextOrFile, nil
	}

	// Rule 2: A file must be relative to the repository
	baseDir, err := os.Getwd()
	if err != nil {
		return "", err
	}
	filePath, err := securejoin.SecureJoin(baseDir, opts.ExperimentalNotesTextOrFile)
	if err != nil {
		return "", err
	}

	b, err := os.ReadFile(filePath)
	if err != nil {
		// Rule 3: fallback to using the value as text
		return opts.ExperimentalNotesTextOrFile, nil
	}

	return string(b), nil
}

func createRun(opts *CreateOpts) error {
	client, err := opts.HTTPClient()
	if err != nil {
		return err
	}

	repo, err := opts.BaseRepo()
	if err != nil {
		return err
	}
	color := opts.IO.Color()
	var tag *gitlab.Tag
	var resp *gitlab.Response

	if opts.Ref == "" {
		opts.IO.Log(color.ProgressIcon(), "Validating tag", opts.TagName)
		tag, resp, err = client.Tags.GetTag(repo.FullName(), opts.TagName)
		if err != nil && resp != nil && resp.StatusCode != http.StatusNotFound {
			return cmdutils.WrapError(err, "could not fetch tag")
		}
		if tag == nil && resp != nil && resp.StatusCode == http.StatusNotFound {
			opts.IO.Log(color.DotWarnIcon(), "Tag does not exist.")
			opts.IO.Log(color.DotWarnIcon(), "No ref provided. Creating the tag from the latest state of the default branch.")
			project, err := repo.Project(client)
			if err == nil {
				opts.IO.Logf("%s using default branch %q as ref\n", color.ProgressIcon(), project.DefaultBranch)
				opts.Ref = project.DefaultBranch
			}
		}
		// new line
		opts.IO.Log()
	}

	if opts.IO.PromptEnabled() && !opts.NoteProvided {
		editorCommand, err := cmdutils.GetEditor(opts.Config)
		if err != nil {
			return err
		}

		var tagDescription string
		var generatedChangelog string
		if tag == nil {
			tag, _, _ = client.Tags.GetTag(repo.FullName(), opts.TagName)
		}
		if tag != nil {
			tagDescription = tag.Message
		}
		if opts.RepoOverride == "" {
			headRef := opts.TagName
			if tagDescription == "" {
				if opts.Ref != "" {
					headRef = opts.Ref
					brCfg := git.ReadBranchConfig(opts.Ref)
					if brCfg.MergeRef != "" {
						headRef = brCfg.MergeRef
					}
				} else {
					headRef = "HEAD"
				}
			}

			if prevTag, err := detectPreviousTag(headRef); err == nil {
				commits, _ := changelogForRange(fmt.Sprintf("%s..%s", prevTag, headRef))
				generatedChangelog = generateChangelog(commits)
			}
		}

		editorOptions := []string{"Write my own."}
		if generatedChangelog != "" {
			editorOptions = append(editorOptions, "Write using the commit log as a template.")
		}
		if tagDescription != "" {
			editorOptions = append(editorOptions, "Write using the Git tag message as the template.")
		}
		editorOptions = append(editorOptions, "Leave blank.")

		qs := []*survey.Question{
			{
				Name: "name",
				Prompt: &survey.Input{
					Message: "Release title (optional)",
					Default: opts.Name,
				},
			},
			{
				Name: "releaseNotesAction",
				Prompt: &survey.Select{
					Message: "Release notes",
					Options: editorOptions,
				},
			},
		}
		err = prompt.Ask(qs, opts)
		if err != nil {
			return fmt.Errorf("could not prompt: %w", err)
		}

		var openEditor bool
		var editorContents string

		switch opts.ReleaseNotesAction {
		case "Write my own.":
			openEditor = true
		case "Write using commit log as template.":
			openEditor = true
			editorContents = generatedChangelog
		case "Write using git tag message as template.":
			openEditor = true
			editorContents = tagDescription
		case "Leave blank.":
			openEditor = false
		default:
			return fmt.Errorf("invalid action: %v", opts.ReleaseNotesAction)
		}

		if openEditor {
			txt, err := surveyext.Edit(editorCommand, "*.md", editorContents, opts.IO.In, opts.IO.StdOut, opts.IO.StdErr, nil)
			if err != nil {
				return err
			}
			opts.Notes = txt
		}
	}
	start := time.Now()

	opts.IO.Logf("%s Creating or updating release %s=%s %s=%s\n",
		color.ProgressIcon(),
		color.Blue("repo"), repo.FullName(),
		color.Blue("tag"), opts.TagName)

	release, resp, err := client.Releases.GetRelease(repo.FullName(), opts.TagName)
	if err != nil && (resp == nil || (resp.StatusCode != http.StatusForbidden && resp.StatusCode != http.StatusNotFound)) {
		return releaseFailedErr(err, start)
	}

	var releasedAt time.Time

	if opts.ReleasedAt != "" {
		// Parse the releasedAt to the expected format of the API
		// From the API docs "Expected in ISO 8601 format (2019-03-15T08:00:00Z)".
		releasedAt, err = time.Parse(time.RFC3339[:len(opts.ReleasedAt)], opts.ReleasedAt)
		if err != nil {
			return releaseFailedErr(err, start)
		}
	}

	if opts.Name == "" {
		opts.Name = opts.TagName
	}

	if (resp.StatusCode == http.StatusForbidden || resp.StatusCode == http.StatusNotFound) || release == nil {
		createOpts := &gitlab.CreateReleaseOptions{
			Name:    &opts.Name,
			TagName: &opts.TagName,
		}

		if opts.Notes != "" {
			createOpts.Description = &opts.Notes
		}

		if opts.Ref != "" {
			createOpts.Ref = &opts.Ref
		}

		if opts.TagMessage != "" {
			createOpts.TagMessage = &opts.TagMessage
		}

		if opts.ReleasedAt != "" {
			createOpts.ReleasedAt = &releasedAt
		}

		if len(opts.Milestone) > 0 {
			createOpts.Milestones = &opts.Milestone
		}

		release, _, err = client.Releases.CreateRelease(repo.FullName(), createOpts)
		if err != nil {
			return releaseFailedErr(err, start)
		}
		opts.IO.Logf("%s Release created:\t%s=%s\n", color.GreenCheck(),
			color.Blue("url"), release.Links.Self)
	} else {
		if opts.NoUpdate {
			return releaseFailedErr(fmt.Errorf("release for tag %q already exists and --no-update flag was specified", opts.TagName), start)
		}

		updateOpts := &gitlab.UpdateReleaseOptions{
			Name: &opts.Name,
		}
		if opts.Notes != "" {
			updateOpts.Description = &opts.Notes
		}

		if opts.ReleasedAt != "" {
			updateOpts.ReleasedAt = &releasedAt
		}

		if len(opts.Milestone) > 0 {
			updateOpts.Milestones = &opts.Milestone
		}

		release, _, err = client.Releases.UpdateRelease(repo.FullName(), opts.TagName, updateOpts)
		if err != nil {
			return releaseFailedErr(err, start)
		}

		opts.IO.Logf("%s Release updated\t%s=%s\n", color.GreenCheck(),
			color.Blue("url"), release.Links.Self)
	}

	// upload files and create asset links
	err = releaseutils.CreateReleaseAssets(opts.IO, client, opts.AssetFiles, opts.AssetLinks, repo.FullName(), release.TagName)
	if err != nil {
		return releaseFailedErr(err, start)
	}

	if opts.NoCloseMilestone {
		opts.IO.Logf("%s Skipping closing milestones\n", color.GreenCheck())
	} else {
		if len(opts.Milestone) > 0 {
			// close all associated milestones
			for _, milestone := range opts.Milestone {
				opts.IO.StartSpinner("Closing milestone %q", milestone)
				err := closeMilestone(opts, milestone)
				opts.IO.StopSpinner("")
				if err != nil {
					opts.IO.Log(color.FailedIcon(), err.Error())
				} else {
					opts.IO.Logf("%s Closed milestone %q\n", color.GreenCheck(), milestone)
				}
			}
		}
	}
	opts.IO.Logf(color.Bold("%s Release succeeded after %0.2f seconds.\n"), color.GreenCheck(), time.Since(start).Seconds())

	if opts.PublishToCatalog {
		err = catalog.Publish(opts.IO, client, repo.FullName(), release.TagName)
		if err != nil {
			return cmdutils.WrapError(err, "failed to publish the release to the GitLab CI/CD catalog.")
		}
	}

	return nil
}

func releaseFailedErr(err error, start time.Time) error {
	return cmdutils.WrapError(err, fmt.Sprintf("release failed after %0.2f seconds.", time.Since(start).Seconds()))
}

func getMilestoneByTitle(c *CreateOpts, title string) (*gitlab.Milestone, error) {
	opts := &gitlab.ListMilestonesOptions{
		Title: &title,
	}

	client, err := c.HTTPClient()
	if err != nil {
		return nil, err
	}

	repo, err := c.BaseRepo()
	if err != nil {
		return nil, err
	}

	for {
		milestones, resp, err := client.Milestones.ListMilestones(repo.FullName(), opts)
		if err != nil {
			return nil, err
		}

		for _, milestone := range milestones {
			if milestone != nil && milestone.Title == title {
				return milestone, nil
			}
		}

		if resp.NextPage == 0 {
			break
		}

		opts.Page = resp.NextPage
	}

	return nil, nil
}

// CloseMilestone closes a given milestone.
func closeMilestone(c *CreateOpts, title string) error {
	client, err := c.HTTPClient()
	if err != nil {
		return err
	}

	repo, err := c.BaseRepo()
	if err != nil {
		return err
	}

	milestone, err := getMilestoneByTitle(c, title)
	if err != nil {
		return err
	}

	if milestone == nil {
		return fmt.Errorf("could not find milestone: %q", title)
	}

	closeStateEvent := "close"

	opts := &gitlab.UpdateMilestoneOptions{
		Description: &milestone.Description,
		DueDate:     milestone.DueDate,
		StartDate:   milestone.StartDate,
		StateEvent:  &closeStateEvent,
		Title:       &milestone.Title,
	}

	_, _, err = client.Milestones.UpdateMilestone(
		repo.FullName(),
		milestone.ID,
		opts,
	)

	return err
}

func detectPreviousTag(headRef string) (string, error) {
	cmd := git.GitCommand("describe", "--tags", "--abbrev=0", fmt.Sprintf("%s^", headRef))
	b, err := run.PrepareCmd(cmd).Output()
	return strings.TrimSpace(string(b)), err
}

type logEntry struct {
	Subject string
	Body    string
}

func changelogForRange(refRange string) ([]logEntry, error) {
	cmd := git.GitCommand("-c", "log.ShowSignature=false", "log", "--first-parent", "--reverse", "--pretty=format:%B%x00", refRange)

	b, err := run.PrepareCmd(cmd).Output()
	if err != nil {
		return nil, err
	}

	var entries []logEntry
	for _, cb := range bytes.Split(b, []byte{'\000'}) {
		c := strings.ReplaceAll(string(cb), "\r\n", "\n")
		c = strings.TrimPrefix(c, "\n")
		if c == "" {
			continue
		}
		parts := strings.SplitN(c, "\n\n", 2)
		var body string
		subject := strings.ReplaceAll(parts[0], "\n", " ")
		if len(parts) > 1 {
			body = parts[1]
		}
		entries = append(entries, logEntry{
			Subject: subject,
			Body:    body,
		})
	}

	return entries, nil
}

func generateChangelog(commits []logEntry) string {
	var parts []string
	for _, c := range commits {
		parts = append(parts, fmt.Sprintf("* %s", c.Subject))
		if c.Body != "" {
			parts = append(parts, utils.Indent(c.Body, "  "))
		}
	}
	return strings.Join(parts, "\n\n")
}
