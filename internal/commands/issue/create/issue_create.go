package create

import (
	"errors"
	"fmt"
	"net/url"
	"os"
	"strconv"
	"strings"

	"gitlab.com/gitlab-org/cli/internal/iostreams"

	"github.com/MakeNowJust/heredoc/v2"
	"gitlab.com/gitlab-org/cli/internal/config"
	"gitlab.com/gitlab-org/cli/internal/glrepo"
	"gitlab.com/gitlab-org/cli/internal/utils"

	"github.com/AlecAivazis/survey/v2"
	"github.com/spf13/cobra"
	gitlab "gitlab.com/gitlab-org/api/client-go"
	"gitlab.com/gitlab-org/cli/internal/api"
	"gitlab.com/gitlab-org/cli/internal/cmdutils"
	"gitlab.com/gitlab-org/cli/internal/commands/issue/issueutils"
	"gitlab.com/gitlab-org/cli/internal/prompt"
	"gitlab.com/gitlab-org/cli/internal/recovery"
)

var createIssue = func(client *gitlab.Client, projectID any, opts *gitlab.CreateIssueOptions) (*gitlab.Issue, error) {
	issue, _, err := client.Issues.CreateIssue(projectID, opts)
	if err != nil {
		return nil, err
	}

	return issue, nil
}

type options struct {
	Title       string   `json:"title,omitempty"`
	Description string   `json:"description,omitempty"`
	Labels      []string `json:"labels,omitempty"`
	Assignees   []string `json:"assignees,omitempty"`

	Weight        int    `json:"weight,omitempty"`
	Milestone     int    `json:"milestone,omitempty"`
	LinkedMR      int    `json:"linked_mr,omitempty"`
	LinkedIssues  []int  `json:"linked_issues,omitempty"`
	IssueLinkType string `json:"issue_link_type,omitempty"`
	TimeEstimate  string `json:"time_estimate,omitempty"`
	TimeSpent     string `json:"time_spent,omitempty"`
	EpicID        int    `json:"epic_id,omitempty"`
	DueDate       string `json:"due_date,omitempty"`

	MilestoneFlag string `json:"milestone_flag"`

	IsConfidential bool `json:"is_confidential,omitempty"`

	noEditor      bool
	isInteractive bool
	yes           bool
	web           bool
	recover       bool

	io         *iostreams.IOStreams
	baseRepo   func() (glrepo.Interface, error)
	httpClient func() (*gitlab.Client, error)
	remotes    func() (glrepo.Remotes, error)
	config     func() config.Config

	baseProject *gitlab.Project
}

func NewCmdCreate(f cmdutils.Factory) *cobra.Command {
	opts := &options{
		io:         f.IO(),
		baseRepo:   f.BaseRepo,
		httpClient: f.HttpClient,
		remotes:    f.Remotes,
		config:     f.Config,
	}
	issueCreateCmd := &cobra.Command{
		Use:     "create [flags]",
		Short:   `Create an issue.`,
		Long:    ``,
		Aliases: []string{"new"},
		Example: heredoc.Doc(`
			$ glab issue create
			$ glab issue new
			$ glab issue create -m release-2.0.0 -t "we need this feature" --label important
			$ glab issue new -t "Fix CVE-YYYY-XXXX" -l security --linked-mr 123
			$ glab issue create -m release-1.0.1 -t "security fix" --label security --web --recover
		`),
		Args: cobra.ExactArgs(0),
		RunE: func(cmd *cobra.Command, args []string) error {
			apiClient, err := opts.httpClient()
			if err != nil {
				return err
			}

			repo, err := opts.baseRepo()
			if err != nil {
				return err
			}
			hasTitle := cmd.Flags().Changed("title")
			hasDescription := cmd.Flags().Changed("description")

			// disable interactive mode if title and description are explicitly defined
			opts.isInteractive = !(hasTitle && hasDescription)

			if opts.isInteractive && !opts.io.PromptEnabled() {
				return &cmdutils.FlagError{Err: errors.New("'--title' and '--description' required for non-interactive mode.")}
			}

			// Remove this once --yes does more than just skip the prompts that --web happen to skip
			// by design
			if opts.yes && opts.web {
				return &cmdutils.FlagError{Err: errors.New("'--web' already skips all prompts currently skipped by '--yes'.")}
			}

			opts.baseProject, err = api.GetProject(apiClient, repo.FullName())
			if err != nil {
				return err
			}

			if !opts.baseProject.IssuesEnabled { //nolint:staticcheck
				fmt.Fprintf(opts.io.StdErr, "Issues are disabled for project %q or require project membership. ", opts.baseProject.PathWithNamespace)
				fmt.Fprintf(opts.io.StdErr, "Make sure issues are enabled for the %q project, and if required, you are a member of the project.\n",
					opts.baseProject.PathWithNamespace)
				return cmdutils.SilentError
			}

			if err := createRun(opts); err != nil {
				// always save options to file
				recoverErr := createRecoverSaveFile(repo.FullName(), opts)
				if recoverErr != nil {
					fmt.Fprintf(opts.io.StdErr, "Could not create recovery file: %v", recoverErr)
				}

				return err
			}

			return nil
		},
	}
	issueCreateCmd.Flags().StringVarP(&opts.Title, "title", "t", "", "Issue title.")
	issueCreateCmd.Flags().StringVarP(&opts.Description, "description", "d", "", "Issue description.")
	issueCreateCmd.Flags().StringSliceVarP(&opts.Labels, "label", "l", []string{}, "Add label by name. Multiple labels should be comma-separated.")
	issueCreateCmd.Flags().StringSliceVarP(&opts.Assignees, "assignee", "a", []string{}, "Assign issue to people by their `usernames`.")
	issueCreateCmd.Flags().StringVarP(&opts.MilestoneFlag, "milestone", "m", "", "The global ID or title of a milestone to assign.")
	issueCreateCmd.Flags().BoolVarP(&opts.IsConfidential, "confidential", "c", false, "Set an issue to be confidential. (default false)")
	issueCreateCmd.Flags().IntVarP(&opts.LinkedMR, "linked-mr", "", 0, "The IID of a merge request in which to resolve all issues.")
	issueCreateCmd.Flags().IntVarP(&opts.Weight, "weight", "w", 0, "Issue weight. Valid values are greater than or equal to 0.")
	issueCreateCmd.Flags().BoolVarP(&opts.noEditor, "no-editor", "", false, "Don't open editor to enter a description. If set to true, uses prompt. (default false)")
	issueCreateCmd.Flags().BoolVarP(&opts.yes, "yes", "y", false, "Don't prompt for confirmation to submit the issue.")
	issueCreateCmd.Flags().BoolVar(&opts.web, "web", false, "Continue issue creation with web interface.")
	issueCreateCmd.Flags().IntSliceVarP(&opts.LinkedIssues, "linked-issues", "", []int{}, "The IIDs of issues that this issue links to.")
	issueCreateCmd.Flags().StringVarP(&opts.IssueLinkType, "link-type", "", "relates_to", "Type for the issue link")
	issueCreateCmd.Flags().StringVarP(&opts.TimeEstimate, "time-estimate", "e", "", "Set time estimate for the issue.")
	issueCreateCmd.Flags().StringVarP(&opts.TimeSpent, "time-spent", "s", "", "Set time spent for the issue.")
	issueCreateCmd.Flags().BoolVar(&opts.recover, "recover", false, "Save the options to a file if the issue fails to be created. If the file exists, the options will be loaded from the recovery file. (EXPERIMENTAL.)")
	issueCreateCmd.Flags().IntVarP(&opts.EpicID, "epic", "", 0, "ID of the epic to add the issue to.")
	issueCreateCmd.Flags().StringVarP(&opts.DueDate, "due-date", "", "", "A date in 'YYYY-MM-DD' format.")

	return issueCreateCmd
}

var createRun = func(opts *options) error {
	apiClient, err := opts.httpClient()
	if err != nil {
		return err
	}

	repo, err := opts.baseRepo()
	if err != nil {
		return err
	}

	var templateName string
	var templateContents string

	issueCreateOpts := &gitlab.CreateIssueOptions{}

	if opts.MilestoneFlag != "" {
		opts.Milestone, err = cmdutils.ParseMilestone(apiClient, repo, opts.MilestoneFlag)
		if err != nil {
			return err
		}
	}

	if opts.recover {
		if err := recovery.FromFile(repo.FullName(), "issue.json", opts); err != nil {
			// if the file to recover doesn't exist, we can just ignore the error and move on
			if !errors.Is(err, os.ErrNotExist) {
				fmt.Fprintf(opts.io.StdErr, "Failed to recover from file: %v", err)
			}
		} else {
			fmt.Fprintln(opts.io.StdOut, "Recovered create options from file.")
		}
	}

	if opts.isInteractive {
		if opts.Description == "" {
			if opts.noEditor {
				err = prompt.AskMultiline(&opts.Description, "description", "Description:", "")
				if err != nil {
					return err
				}
			} else {

				templateResponse := struct {
					Index int
				}{}
				templateNames, err := cmdutils.ListGitLabTemplates(cmdutils.IssueTemplate)
				if err != nil {
					return fmt.Errorf("error getting templates: %w", err)
				}

				templateNames = append(templateNames, "Open a blank issue")

				selectQs := []*survey.Question{
					{
						Name: "index",
						Prompt: &survey.Select{
							Message: "Choose a template",
							Options: templateNames,
						},
					},
				}

				if err := prompt.Ask(selectQs, &templateResponse); err != nil {
					return fmt.Errorf("could not prompt: %w", err)
				}
				if templateResponse.Index != len(templateNames) {
					templateName = templateNames[templateResponse.Index]
					templateContents, err = cmdutils.LoadGitLabTemplate(cmdutils.IssueTemplate, templateName)
					if err != nil {
						return fmt.Errorf("failed to get template contents: %w", err)
					}
				}
			}
		}
		if opts.Title == "" {
			err = prompt.AskQuestionWithInput(&opts.Title, "title", "Title", "", true)
			if err != nil {
				return err
			}
		}
		if opts.Description == "" {
			if opts.noEditor {
				err = prompt.AskMultiline(&opts.Description, "description", "Description:", "")
				if err != nil {
					return err
				}
			} else {
				editor, err := cmdutils.GetEditor(opts.config)
				if err != nil {
					return err
				}
				err = cmdutils.EditorPrompt(&opts.Description, "Description", templateContents, editor)
				if err != nil {
					return err
				}
			}
		}
	} else if opts.Title == "" {
		return fmt.Errorf("title can't be blank")
	}

	var action cmdutils.Action

	// submit without prompting for non interactive mode
	if !opts.isInteractive || opts.yes {
		action = cmdutils.SubmitAction
	}

	if opts.web {
		action = cmdutils.PreviewAction
	}

	if action == cmdutils.NoAction {
		action, err = cmdutils.ConfirmSubmission(true, true)
		if err != nil {
			return fmt.Errorf("unable to confirm: %w", err)
		}
	}

	if action == cmdutils.AddMetadataAction {
		var metadataActions []cmdutils.Action

		metadataActions, err = cmdutils.PickMetadata()
		if err != nil {
			return fmt.Errorf("failed to pick metadata to add: %w", err)
		}

		remotes, err := opts.remotes()
		if err != nil {
			return err
		}
		repoRemote, err := remotes.FindByRepo(repo.RepoOwner(), repo.RepoName())
		if err != nil {
			// when the base repo is overridden with --repo flag, it is likely it has no
			// remote set for the current working git dir which will error.
			// Init a new remote without actually adding a new git remote
			repoRemote = &glrepo.Remote{
				Repo: repo,
			}
		}

		for _, x := range metadataActions {
			if x == cmdutils.AddLabelAction {
				err = cmdutils.LabelsPrompt(&opts.Labels, apiClient, repoRemote)
				if err != nil {
					return err
				}

			}
			if x == cmdutils.AddAssigneeAction {
				// Involve only reporters and up, in the future this might be expanded to `guests`
				// but that might hit the `100` limit for projects with large amounts of collaborators
				err = cmdutils.UsersPrompt(&opts.Assignees, apiClient, repoRemote, opts.io, 20, "assignees")
				if err != nil {
					return err
				}
			}
			if x == cmdutils.AddMilestoneAction {
				err = cmdutils.MilestonesPrompt(&opts.Milestone, apiClient, repoRemote, opts.io)
				if err != nil {
					return err
				}

			}
		}

		// Ask the user again but don't permit AddMetadata a second time
		action, err = cmdutils.ConfirmSubmission(true, false)
		if err != nil {
			return err
		}
	}

	if action == cmdutils.CancelAction {
		fmt.Fprintln(opts.io.StdErr, "Discarded.")
		return nil
	}

	if action == cmdutils.PreviewAction {
		return previewIssue(opts)
	}

	if action == cmdutils.SubmitAction {
		issueCreateOpts.Title = gitlab.Ptr(opts.Title)
		issueCreateOpts.Labels = (*gitlab.LabelOptions)(&opts.Labels)
		issueCreateOpts.Description = &opts.Description
		if opts.IsConfidential {
			issueCreateOpts.Confidential = gitlab.Ptr(opts.IsConfidential)
		}
		if opts.Weight != 0 {
			issueCreateOpts.Weight = gitlab.Ptr(opts.Weight)
		}
		if opts.LinkedMR != 0 {
			issueCreateOpts.MergeRequestToResolveDiscussionsOf = gitlab.Ptr(opts.LinkedMR)
		}
		if opts.Milestone != 0 {
			issueCreateOpts.MilestoneID = gitlab.Ptr(opts.Milestone)
		}
		if opts.EpicID != 0 {
			issueCreateOpts.EpicID = gitlab.Ptr(opts.EpicID)
		}
		if opts.DueDate != "" {
			dueDate, err := gitlab.ParseISOTime(opts.DueDate)
			if err != nil {
				return err
			}
			issueCreateOpts.DueDate = gitlab.Ptr(dueDate)
		}

		if len(opts.Assignees) > 0 {
			users, err := api.UsersByNames(apiClient, opts.Assignees)
			if err != nil {
				return err
			}
			issueCreateOpts.AssigneeIDs = cmdutils.IDsFromUsers(users)
		}
		fmt.Fprintln(opts.io.StdErr, "- Creating issue in", repo.FullName())
		issue, err := createIssue(apiClient, repo.FullName(), issueCreateOpts)
		if err != nil {
			return err
		}
		if err := postCreateActions(apiClient, issue, opts, repo); err != nil {
			return err
		}

		fmt.Fprintln(opts.io.StdOut, issueutils.DisplayIssue(opts.io.Color(), issue, opts.io.IsaTTY))
		return nil
	}

	return errors.New("expected to cancel, preview in browser, add metadata, or submit")
}

func postCreateActions(apiClient *gitlab.Client, issue *gitlab.Issue, opts *options, repo glrepo.Interface) error {
	if len(opts.LinkedIssues) > 0 {
		for _, targetIssueIID := range opts.LinkedIssues {
			fmt.Fprintln(opts.io.StdErr, "- Linking to issue ", targetIssueIID)
			issueLink, _, err := apiClient.IssueLinks.CreateIssueLink(repo.FullName(), issue.IID, &gitlab.CreateIssueLinkOptions{
				TargetIssueIID: gitlab.Ptr(strconv.Itoa(targetIssueIID)),
				LinkType:       gitlab.Ptr(opts.IssueLinkType),
			})
			if err != nil {
				return err
			}
			issue = issueLink.SourceIssue
		}
	}
	if opts.TimeEstimate != "" {
		fmt.Fprintln(opts.io.StdErr, "- Adding time estimate ", opts.TimeEstimate)
		_, _, err := apiClient.Issues.SetTimeEstimate(repo.FullName(), issue.IID, &gitlab.SetTimeEstimateOptions{Duration: gitlab.Ptr(opts.TimeEstimate)})
		if err != nil {
			return err
		}
	}
	if opts.TimeSpent != "" {
		fmt.Fprintln(opts.io.StdErr, "- Adding time spent ", opts.TimeSpent)
		_, _, err := apiClient.Issues.AddSpentTime(repo.FullName(), issue.IID, &gitlab.AddSpentTimeOptions{Duration: gitlab.Ptr(opts.TimeSpent)})
		if err != nil {
			return err
		}
	}
	return nil
}

func previewIssue(opts *options) error {
	repo, err := opts.baseRepo()
	if err != nil {
		return err
	}

	cfg := opts.config()

	openURL, err := generateIssueWebURL(opts)
	if err != nil {
		return err
	}

	if opts.io.IsOutputTTY() {
		fmt.Fprintf(opts.io.StdErr, "Opening %s in your browser.\n", utils.DisplayURL(openURL))
	}
	browser, _ := cfg.Get(repo.RepoHost(), "browser")
	return utils.OpenInBrowser(openURL, browser)
}

func generateIssueWebURL(opts *options) (string, error) {
	description := opts.Description

	if len(opts.Labels) > 0 {
		// this uses the slash commands to add labels to the description
		// See https://docs.gitlab.com/user/project/quick_actions/
		// See also https://gitlab.com/gitlab-org/gitlab-foss/-/issues/19731#note_32550046
		description += "\n/label"
		for _, label := range opts.Labels {
			description += fmt.Sprintf(" ~%q", label)
		}
	}
	if len(opts.Assignees) > 0 {
		// this uses the slash commands to add assignees to the description
		description += fmt.Sprintf("\n/assign %s", strings.Join(opts.Assignees, ", "))
	}
	if opts.Milestone != 0 {
		// this uses the slash commands to add milestone to the description
		description += fmt.Sprintf("\n/milestone %%%d", opts.Milestone)
	}
	if opts.Weight != 0 {
		// this uses the slash commands to add weight to the description
		description += fmt.Sprintf("\n/weight %d", opts.Weight)
	}
	if opts.IsConfidential {
		// this uses the slash commands to add confidential to the description
		description += "\n/confidential"
	}

	u, err := url.Parse(opts.baseProject.WebURL)
	if err != nil {
		return "", err
	}
	u.Path += "/-/issues/new"

	q := u.Query()
	q.Set("issue[title]", opts.Title)
	q.Add("issue[description]", description)
	u.RawQuery = q.Encode()

	return u.String(), nil
}

// createRecoverSaveFile will try save the issue create options to a file
func createRecoverSaveFile(repoName string, opts *options) error {
	recoverFile, err := recovery.CreateFile(repoName, "issue.json", opts)
	if err != nil {
		return err
	}

	fmt.Fprintf(opts.io.StdErr, "Failed to create issue. Created recovery file: %s\nRun the command again with the '--recover' option to retry", recoverFile)
	return nil
}
