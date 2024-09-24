package create

import (
	"errors"
	"fmt"
	"net/url"
	"os"
	"regexp"
	"strconv"
	"strings"

	"github.com/AlecAivazis/survey/v2"
	"gitlab.com/gitlab-org/cli/commands/issue/issueutils"
	"gitlab.com/gitlab-org/cli/pkg/prompt"

	"gitlab.com/gitlab-org/cli/pkg/iostreams"

	"github.com/MakeNowJust/heredoc/v2"
	"gitlab.com/gitlab-org/cli/internal/config"
	"gitlab.com/gitlab-org/cli/internal/glrepo"
	"gitlab.com/gitlab-org/cli/internal/recovery"

	"github.com/spf13/cobra"
	"github.com/xanzy/go-gitlab"
	"gitlab.com/gitlab-org/cli/api"
	"gitlab.com/gitlab-org/cli/commands/cmdutils"
	"gitlab.com/gitlab-org/cli/commands/mr/mrutils"
	"gitlab.com/gitlab-org/cli/pkg/git"
	"gitlab.com/gitlab-org/cli/pkg/utils"
)

type CreateOpts struct {
	Title                 string   `json:"title,omitempty"`
	Description           string   `json:"description,omitempty"`
	SourceBranch          string   `json:"source_branch,omitempty"`
	TargetBranch          string   `json:"target_branch,omitempty"`
	TargetTrackingBranch  string   `json:"target_tracking_branch,omitempty"`
	Labels                []string `json:"labels,omitempty"`
	Assignees             []string `json:"assignees,omitempty"`
	Reviewers             []string `json:"reviewers,omitempty"`
	Milestone             int      `json:"milestone,omitempty"`
	MilestoneFlag         string   `json:"milestone_flag,omitempty"`
	MRCreateTargetProject string   `json:"mr_create_target_project,omitempty"`

	RelatedIssue    string `json:"related_issue,omitempty"`
	CopyIssueLabels bool   `json:"copy_issue_labels,omitempty"`

	CreateSourceBranch bool `json:"create_source_branch,omitempty"`
	RemoveSourceBranch bool `json:"remove_source_branch,omitempty"`
	AllowCollaboration bool `json:"allow_collaboration,omitempty"`
	SquashBeforeMerge  bool `json:"squash_before_merge,omitempty"`

	Autofill       bool `json:"autofill,omitempty"`
	FillCommitBody bool `json:"fill_commit_body,omitempty"`
	IsDraft        bool `json:"is_draft,omitempty"`
	IsWIP          bool `json:"is_wip,omitempty"`
	ShouldPush     bool `json:"should_push,omitempty"`

	NoEditor      bool `json:"-"`
	IsInteractive bool `json:"-"`
	Yes           bool `json:"-"`
	Web           bool `json:"-"`
	Recover       bool `json:"-"`
	Signoff       bool `json:"-"`

	IO       *iostreams.IOStreams             `json:"-"`
	Branch   func() (string, error)           `json:"-"`
	Remotes  func() (glrepo.Remotes, error)   `json:"-"`
	Lab      func() (*gitlab.Client, error)   `json:"-"`
	Config   func() (config.Config, error)    `json:"-"`
	BaseRepo func() (glrepo.Interface, error) `json:"-"`
	HeadRepo func() (glrepo.Interface, error) `json:"-"`

	// SourceProject is the Project we create the merge request in and where we push our branch
	// it is the project we have permission to push so most likely one's fork
	SourceProject *gitlab.Project `json:"source_project,omitempty"`
	// TargetProject is the one we query for changes between our branch and the target branch
	// it is the one we merge request will appear in
	TargetProject *gitlab.Project `json:"target_project,omitempty"`
}

func NewCmdCreate(f *cmdutils.Factory, runE func(opts *CreateOpts) error) *cobra.Command {
	opts := &CreateOpts{
		IO:       f.IO,
		Branch:   f.Branch,
		Remotes:  f.Remotes,
		Config:   f.Config,
		HeadRepo: ResolvedHeadRepo(f),
	}

	mrCreateCmd := &cobra.Command{
		Use:     "create",
		Short:   `Create a new merge request.`,
		Long:    ``,
		Aliases: []string{"new"},
		Example: heredoc.Doc(`
			glab mr new
			glab mr create -a username -t "fix annoying bug"
			glab mr create -f --draft --label RFC
			glab mr create --fill --web
			glab mr create --fill --fill-commit-body --yes
		`),
		Args: cobra.ExactArgs(0),
		PreRun: func(cmd *cobra.Command, args []string) {
			repoOverride, _ := cmd.Flags().GetString("head")
			if repoFromEnv := os.Getenv("GITLAB_HEAD_REPO"); repoOverride == "" && repoFromEnv != "" {
				repoOverride = repoFromEnv
			}
			if repoOverride != "" {
				_ = headRepoOverride(opts, repoOverride)
			}
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			// support `-R, --repo` override
			//
			// NOTE: it is important to assign the BaseRepo and HTTPClient in RunE because
			// they are overridden in a PersistentRun hook (when `-R, --repo` is specified)
			// which runs before RunE is executed
			opts.BaseRepo = f.BaseRepo
			opts.Lab = f.HttpClient

			hasTitle := cmd.Flags().Changed("title")
			hasDescription := cmd.Flags().Changed("description")

			// disable interactive mode if title and description are explicitly defined
			opts.IsInteractive = !(hasTitle && hasDescription)

			if hasTitle && hasDescription && opts.Autofill {
				return &cmdutils.FlagError{
					Err: errors.New("usage of --title and --description overrides --fill."),
				}
			}
			if opts.IsInteractive && !opts.IO.PromptEnabled() && !opts.Autofill {
				return &cmdutils.FlagError{Err: errors.New("--title or --fill required for non-interactive mode.")}
			}
			if cmd.Flags().Changed("wip") && cmd.Flags().Changed("draft") {
				return &cmdutils.FlagError{Err: errors.New("specify --draft.")}
			}
			if !opts.Autofill && opts.FillCommitBody {
				return &cmdutils.FlagError{Err: errors.New("--fill-commit-body should be used with --fill.")}
			}
			// Remove this once --yes does more than just skip the prompts that --web happen to skip
			// by design
			if opts.Yes && opts.Web {
				return &cmdutils.FlagError{Err: errors.New("--web already skips all prompts currently skipped by --yes.")}
			}

			if opts.CopyIssueLabels && opts.RelatedIssue == "" {
				return &cmdutils.FlagError{Err: errors.New("--copy-issue-labels can only be used with --related-issue.")}
			}

			if runE != nil {
				return runE(opts)
			}

			if err := createRun(opts); err != nil {
				// always save options to file
				recoverErr := createRecoverSaveFile(opts)
				if recoverErr != nil {
					fmt.Fprintf(opts.IO.StdErr, "Could not create recovery file: %v", recoverErr)
				}

				return err
			}

			return nil
		},
	}
	mrCreateCmd.Flags().BoolVarP(&opts.Autofill, "fill", "f", false, "Do not prompt for title or description, and just use commit info. Sets `push` to `true`, and pushes the branch.")
	mrCreateCmd.Flags().BoolVarP(&opts.FillCommitBody, "fill-commit-body", "", false, "Fill description with each commit body when multiple commits. Can only be used with --fill.")
	mrCreateCmd.Flags().BoolVarP(&opts.IsDraft, "draft", "", false, "Mark merge request as a draft.")
	mrCreateCmd.Flags().BoolVarP(&opts.IsWIP, "wip", "", false, "Mark merge request as a draft. Alternative to --draft.")
	mrCreateCmd.Flags().BoolVarP(&opts.ShouldPush, "push", "", false, "Push committed changes after creating merge request. Make sure you have committed changes.")
	mrCreateCmd.Flags().StringVarP(&opts.Title, "title", "t", "", "Supply a title for the merge request.")
	mrCreateCmd.Flags().StringVarP(&opts.Description, "description", "d", "", "Supply a description for the merge request.")
	mrCreateCmd.Flags().StringSliceVarP(&opts.Labels, "label", "l", []string{}, "Add label by name. Multiple labels should be comma-separated.")
	mrCreateCmd.Flags().StringSliceVarP(&opts.Assignees, "assignee", "a", []string{}, "Assign merge request to people by their `usernames`.")
	mrCreateCmd.Flags().StringSliceVarP(&opts.Reviewers, "reviewer", "", []string{}, "Request review from users by their `usernames`.")
	mrCreateCmd.Flags().StringVarP(&opts.SourceBranch, "source-branch", "s", "", "Create a merge request from this branch. Default is the current branch.")
	mrCreateCmd.Flags().StringVarP(&opts.TargetBranch, "target-branch", "b", "", "The target or base branch into which you want your code merged into.")
	mrCreateCmd.Flags().BoolVarP(&opts.CreateSourceBranch, "create-source-branch", "", false, "Create a source branch if it does not exist.")
	mrCreateCmd.Flags().StringVarP(&opts.MilestoneFlag, "milestone", "m", "", "The global ID or title of a milestone to assign.")
	mrCreateCmd.Flags().BoolVarP(&opts.AllowCollaboration, "allow-collaboration", "", false, "Allow commits from other members.")
	mrCreateCmd.Flags().BoolVarP(&opts.RemoveSourceBranch, "remove-source-branch", "", false, "Remove source branch on merge.")
	mrCreateCmd.Flags().BoolVarP(&opts.SquashBeforeMerge, "squash-before-merge", "", false, "Squash commits into a single commit when merging.")
	mrCreateCmd.Flags().BoolVarP(&opts.NoEditor, "no-editor", "", false, "Don't open editor to enter a description. If true, uses prompt. Defaults to false.")
	mrCreateCmd.Flags().StringP("head", "H", "", "Select another head repository using the `OWNER/REPO` or `GROUP/NAMESPACE/REPO` format, the project ID, or the full URL.")
	mrCreateCmd.Flags().BoolVarP(&opts.Yes, "yes", "y", false, "Skip submission confirmation prompt. Use --fill to skip all optional prompts.")
	mrCreateCmd.Flags().BoolVarP(&opts.Web, "web", "w", false, "Continue merge request creation in a browser.")
	mrCreateCmd.Flags().BoolVarP(&opts.CopyIssueLabels, "copy-issue-labels", "", false, "Copy labels from issue to the merge request. Used with --related-issue.")
	mrCreateCmd.Flags().StringVarP(&opts.RelatedIssue, "related-issue", "i", "", "Create a merge request for an issue. If --title is not provided, uses the issue title.")
	mrCreateCmd.Flags().BoolVar(&opts.Recover, "recover", false, "Save the options to a file if the merge request creation fails. If the file exists, the options are loaded from the recovery file. (EXPERIMENTAL.)")
	mrCreateCmd.Flags().BoolVar(&opts.Signoff, "signoff", false, "Append a DCO signoff to the merge request description.")

	mrCreateCmd.Flags().StringVarP(&opts.MRCreateTargetProject, "target-project", "", "", "Add target project by id, OWNER/REPO, or GROUP/NAMESPACE/REPO.")
	_ = mrCreateCmd.Flags().MarkHidden("target-project")
	_ = mrCreateCmd.Flags().MarkDeprecated("target-project", "Use --repo instead.")

	return mrCreateCmd
}

func parseIssue(apiClient *gitlab.Client, opts *CreateOpts) (*gitlab.Issue, error) {
	issue, _, err := issueutils.IssueFromArg(apiClient, opts.BaseRepo, opts.RelatedIssue)
	if err != nil {
		return nil, err
	}

	return issue, nil
}

func createRun(opts *CreateOpts) error {
	out := opts.IO.StdOut
	c := opts.IO.Color()
	mrCreateOpts := &gitlab.CreateMergeRequestOptions{}
	glRepo, err := opts.BaseRepo()
	if err != nil {
		return err
	}

	if opts.Recover {
		if err := recovery.FromFile(glRepo.FullName(), "mr.json", opts); err != nil {
			// if the file to recover doesn't exist, we can just ignore the error and move on
			if !errors.Is(err, os.ErrNotExist) {
				fmt.Fprintf(opts.IO.StdErr, "Failed to recover from file: %v", err)
			}
		} else {
			fmt.Fprintln(opts.IO.StdOut, "Recovered create options from file")
		}
	}

	labClient, err := opts.Lab()
	if err != nil {
		return err
	}

	baseRepo, err := opts.BaseRepo()
	if err != nil {
		return err
	}

	headRepo, err := opts.HeadRepo()
	if err != nil {
		return err
	}

	// only fetch source project if it wasn't saved for recovery
	if opts.SourceProject == nil {
		opts.SourceProject, err = api.GetProject(labClient, headRepo.FullName())
		if err != nil {
			return err
		}
	}

	// only fetch target project if it wasn't saved for recovery
	if opts.TargetProject == nil {
		// if the user set the target_project, get details of the target
		if opts.MRCreateTargetProject != "" {
			opts.TargetProject, err = api.GetProject(labClient, opts.MRCreateTargetProject)
			if err != nil {
				return err
			}
		} else {
			// If both the baseRepo and headRepo are the same then re-use the SourceProject
			if baseRepo.FullName() == headRepo.FullName() {
				opts.TargetProject = opts.SourceProject
			} else {
				// Otherwise assume the user wants to create the merge request against the
				// baseRepo
				opts.TargetProject, err = api.GetProject(labClient, baseRepo.FullName())
				if err != nil {
					return err
				}
			}
		}
	}

	if !opts.TargetProject.MergeRequestsEnabled {
		fmt.Fprintf(opts.IO.StdErr, "Failed to create a merge request for project %q. Please ensure:\n", opts.TargetProject.PathWithNamespace)
		fmt.Fprintf(opts.IO.StdErr, " - You are authenticated with the GitLab CLI.\n")
		fmt.Fprintf(opts.IO.StdErr, " - Merge requests are enabled for this project.\n")
		fmt.Fprintf(opts.IO.StdErr, " - Your role in this project allows you to create merge requests.\n")
		return cmdutils.SilentError
	}

	headRepoRemote, err := repoRemote(opts, headRepo, opts.SourceProject, "glab-head")
	if err != nil {
		return nil
	}

	var baseRepoRemote *glrepo.Remote

	// check if baseRepo is the same as the headRepo and set the remote
	if glrepo.IsSame(baseRepo, headRepo) {
		baseRepoRemote = headRepoRemote
	} else {
		baseRepoRemote, err = repoRemote(opts, baseRepo, opts.TargetProject, "glab-base")
		if err != nil {
			return nil
		}
	}

	if opts.MilestoneFlag != "" {
		opts.Milestone, err = cmdutils.ParseMilestone(labClient, baseRepo, opts.MilestoneFlag)
		if err != nil {
			return err
		}
	}

	if opts.CreateSourceBranch && opts.SourceBranch == "" {
		opts.SourceBranch = utils.ReplaceNonAlphaNumericChars(opts.Title, "-")
	} else if opts.SourceBranch == "" && opts.RelatedIssue == "" {
		opts.SourceBranch, err = opts.Branch()
		if err != nil {
			return err
		}
	}

	if opts.TargetBranch == "" {
		opts.TargetBranch = getTargetBranch(baseRepoRemote)
	}

	if opts.RelatedIssue != "" {
		issue, err := parseIssue(labClient, opts)
		if err != nil {
			return err
		}

		if opts.CopyIssueLabels {
			mrCreateOpts.Labels = (*gitlab.LabelOptions)(&issue.Labels)
		}

		opts.Description += fmt.Sprintf("\n\nCloses #%d", issue.IID)

		if opts.Title == "" {
			opts.Title = fmt.Sprintf("Resolve \"%s\"", issue.Title)
		}

		// MRs created with a related issue will always be created as a draft, same as the UI
		if !opts.IsDraft && !opts.IsWIP {
			opts.IsDraft = true
		}

		if opts.SourceBranch == "" {
			sourceBranch := fmt.Sprintf("%d-%s", issue.IID, utils.ReplaceNonAlphaNumericChars(strings.ToLower(issue.Title), "-"))
			branchOpts := &gitlab.CreateBranchOptions{
				Branch: &sourceBranch,
				Ref:    &opts.TargetBranch,
			}

			_, err = api.CreateBranch(labClient, baseRepo.FullName(), branchOpts)
			if err != nil {
				for branchErr, branchCount := err, 1; branchErr != nil; branchCount++ {
					sourceBranch = fmt.Sprintf("%d-%s-%d", issue.IID, strings.ReplaceAll(strings.ToLower(issue.Title), " ", "-"), branchCount)
					_, branchErr = api.CreateBranch(labClient, baseRepo.FullName(), branchOpts)
				}
			}
			opts.SourceBranch = sourceBranch
		}
	} else {
		opts.TargetTrackingBranch = fmt.Sprintf("%s/%s", baseRepoRemote.Name, opts.TargetBranch)
		if opts.SourceBranch == opts.TargetBranch && glrepo.IsSame(baseRepo, headRepo) {
			fmt.Fprintf(opts.IO.StdErr, "You must be on a different branch other than %q\n", opts.TargetBranch)
			return cmdutils.SilentError
		}

		if opts.Autofill {
			if err = mrBodyAndTitle(opts); err != nil {
				return err
			}
			_, err = api.GetCommit(labClient, baseRepo.FullName(), opts.TargetBranch)
			if err != nil {
				return fmt.Errorf("target branch %s does not exist on remote. Specify target branch with the --target-branch flag",
					opts.TargetBranch)
			}

			opts.ShouldPush = true
		} else if opts.IsInteractive {
			var templateName string
			var templateContents string
			if opts.Description == "" {
				if opts.NoEditor {
					err = prompt.AskMultiline(&opts.Description, "description", "Description:", "")
					if err != nil {
						return err
					}
				} else {
					templateResponse := struct {
						Index int
					}{}
					templateNames, err := cmdutils.ListGitLabTemplates(cmdutils.MergeRequestTemplate)
					if err != nil {
						return fmt.Errorf("error getting templates: %w", err)
					}

					const mrWithCommitsTemplate = "Open a merge request with commit messages."
					const mrEmptyTemplate = "Open a blank merge request."

					templateNames = append(templateNames, mrWithCommitsTemplate)
					templateNames = append(templateNames, mrEmptyTemplate)

					selectQs := []*survey.Question{
						{
							Name: "index",
							Prompt: &survey.Select{
								Message: "Choose a template:",
								Options: templateNames,
							},
						},
					}

					if err := prompt.Ask(selectQs, &templateResponse); err != nil {
						return fmt.Errorf("could not prompt: %w", err)
					}

					templateName = templateNames[templateResponse.Index]
					if templateName == mrWithCommitsTemplate {
						// templateContents should be filled from commit messages
						commits, err := git.Commits(opts.TargetTrackingBranch, opts.SourceBranch)
						if err != nil {
							return fmt.Errorf("failed to get commits: %w", err)
						}
						templateContents, err = mrBody(commits, true)
						if err != nil {
							return err
						}
						if opts.Signoff {
							u, _ := api.CurrentUser(labClient)
							templateContents += "Signed-off-by: " + u.Name + "<" + u.Email + ">"
						}
					} else if templateName == mrEmptyTemplate {
						// blank merge request was choosen, leave templateContents empty
						if opts.Signoff {
							u, _ := api.CurrentUser(labClient)
							templateContents += "Signed-off-by: " + u.Name + "<" + u.Email + ">"
						}
					} else {
						templateContents, err = cmdutils.LoadGitLabTemplate(cmdutils.MergeRequestTemplate, templateName)
						if err != nil {
							return fmt.Errorf("failed to get template contents: %w", err)
						}
					}
				}
			}

			if opts.Title == "" {
				err = prompt.AskQuestionWithInput(&opts.Title, "title", "Title:", "", true)
				if err != nil {
					return err
				}
			}
			if opts.Description == "" {
				if opts.NoEditor {
					err = prompt.AskMultiline(&opts.Description, "description", "Description:", "")
					if err != nil {
						return err
					}
				} else {
					editor, err := cmdutils.GetEditor(opts.Config)
					if err != nil {
						return err
					}
					err = cmdutils.EditorPrompt(&opts.Description, "Description", templateContents, editor)
					if err != nil {
						return err
					}
				}
			}
		}
	}

	if opts.Title == "" {
		return fmt.Errorf("title can't be blank.")
	}

	if opts.IsDraft || opts.IsWIP {
		if opts.IsDraft {
			opts.Title = "Draft: " + opts.Title
		} else {
			opts.Title = "WIP: " + opts.Title
		}
	}
	mrCreateOpts.Title = &opts.Title
	mrCreateOpts.Description = &opts.Description
	mrCreateOpts.SourceBranch = &opts.SourceBranch
	mrCreateOpts.TargetBranch = &opts.TargetBranch

	if opts.AllowCollaboration {
		mrCreateOpts.AllowCollaboration = gitlab.Ptr(true)
	}

	if opts.RemoveSourceBranch {
		mrCreateOpts.RemoveSourceBranch = gitlab.Ptr(true)
	}

	if opts.SquashBeforeMerge {
		mrCreateOpts.Squash = gitlab.Ptr(true)
	}

	if opts.TargetProject != nil {
		mrCreateOpts.TargetProjectID = &opts.TargetProject.ID
	}

	if opts.CreateSourceBranch {
		lb := &gitlab.CreateBranchOptions{
			Branch: &opts.SourceBranch,
			Ref:    &opts.TargetBranch,
		}
		fmt.Fprintln(opts.IO.StdErr, "\nCreating related branch...")
		branch, err := api.CreateBranch(labClient, headRepo.FullName(), lb)
		if err == nil {
			fmt.Fprintln(opts.IO.StdErr, "Branch created: ", branch.WebURL)
		} else {
			fmt.Fprintln(opts.IO.StdErr, "Error creating branch: ", err.Error())
		}
	}

	var action cmdutils.Action

	// submit without prompting for non interactive mode
	if !opts.IsInteractive || opts.Yes {
		action = cmdutils.SubmitAction
	}

	if opts.Web {
		action = cmdutils.PreviewAction
	}

	if action == cmdutils.NoAction {
		action, err = cmdutils.ConfirmSubmission(true, true)
		if err != nil {
			return fmt.Errorf("unable to confirm: %w", err)
		}
	}

	if action == cmdutils.AddMetadataAction {
		metadataOptions := []string{
			"labels",
			"assignees",
			"milestones",
			"reviewers",
		}
		var metadataActions []string

		err := prompt.MultiSelect(&metadataActions, "metadata", "Which metadata types to add?", metadataOptions)
		if err != nil {
			return fmt.Errorf("failed to pick the metadata to add: %w", err)
		}

		for _, x := range metadataActions {
			if x == "labels" {
				err = cmdutils.LabelsPrompt(&opts.Labels, labClient, baseRepoRemote)
				if err != nil {
					return err
				}
			}
			if x == "assignees" {
				// Use minimum permission level 30 (Maintainer) as it is the minimum level
				// to accept a merge request
				err = cmdutils.UsersPrompt(&opts.Assignees, labClient, baseRepoRemote, opts.IO, 30, x)
				if err != nil {
					return err
				}
			}
			if x == "milestones" {
				err = cmdutils.MilestonesPrompt(&opts.Milestone, labClient, baseRepoRemote, opts.IO)
				if err != nil {
					return err
				}
			}
			if x == "reviewers" {
				// Use minimum permission level 30 (Maintainer) as it is the minimum level
				// to accept a merge request
				err = cmdutils.UsersPrompt(&opts.Reviewers, labClient, baseRepoRemote, opts.IO, 30, x)
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

	// This check protects against possibly dereferencing a nil pointer.
	if mrCreateOpts.Labels == nil {
		mrCreateOpts.Labels = &gitlab.LabelOptions{}
	}
	// These actions need to be done here, after the `Add metadata` prompt because
	// they are metadata that can be modified by the prompt
	*mrCreateOpts.Labels = append(*mrCreateOpts.Labels, opts.Labels...)

	if len(opts.Assignees) > 0 {
		users, err := api.UsersByNames(labClient, opts.Assignees)
		if err != nil {
			return err
		}
		mrCreateOpts.AssigneeIDs = cmdutils.IDsFromUsers(users)
	}

	if len(opts.Reviewers) > 0 {
		users, err := api.UsersByNames(labClient, opts.Reviewers)
		if err != nil {
			return err
		}
		mrCreateOpts.ReviewerIDs = cmdutils.IDsFromUsers(users)
	}

	if opts.Milestone != 0 {
		mrCreateOpts.MilestoneID = gitlab.Ptr(opts.Milestone)
	}

	if action == cmdutils.CancelAction {
		fmt.Fprintln(opts.IO.StdErr, "Discarded.")
		return nil
	}

	if err := handlePush(opts, headRepoRemote); err != nil {
		return err
	}

	if action == cmdutils.PreviewAction {
		return previewMR(opts)
	}

	if action == cmdutils.SubmitAction {
		message := "\nCreating merge request for %s into %s in %s\n\n"
		if opts.IsDraft || opts.IsWIP {
			message = "\nCreating draft merge request for %s into %s in %s\n\n"
		}

		fmt.Fprintf(opts.IO.StdErr, message, c.Cyan(opts.SourceBranch), c.Cyan(opts.TargetBranch), baseRepo.FullName())

		// It is intentional that we create against the head repo, it is necessary
		// for cross-repository merge requests
		mr, err := api.CreateMR(labClient, headRepo.FullName(), mrCreateOpts)
		if err != nil {
			return err
		}

		fmt.Fprintln(out, mrutils.DisplayMR(c, mr, opts.IO.IsaTTY))
		return nil
	}

	return errors.New("expected to cancel, preview in browser, or submit.")
}

func mrBody(commits []*git.Commit, fillCommitBody bool) (string, error) {
	var body strings.Builder
	re := regexp.MustCompile(`\r?\n\n`)

	for i := len(commits) - 1; i >= 0; i-- {
		// adds 2 spaces for markdown line wrapping
		fmt.Fprintf(&body, "- %s  \n", commits[i].Title)

		if fillCommitBody {
			commitBody, err := git.CommitBody(commits[i].Sha)
			if err != nil {
				return "", fmt.Errorf("failed to get commit message for %s: %w", commits[i].Sha, err)
			}
			commitBody = re.ReplaceAllString(commitBody, "  \n")
			fmt.Fprintf(&body, "%s\n", commitBody)
		}
	}

	return body.String(), nil
}

func mrBodyAndTitle(opts *CreateOpts) error {
	// TODO: detect forks
	commits, err := git.Commits(opts.TargetTrackingBranch, opts.SourceBranch)
	if err != nil {
		return err
	}
	if len(commits) == 1 {
		if opts.Title == "" {
			opts.Title = commits[0].Title
		}
		if opts.Description == "" {
			body, err := git.CommitBody(commits[0].Sha)
			if err != nil {
				return err
			}
			opts.Description = body
		}
	} else {
		if opts.Title == "" {
			opts.Title = utils.Humanize(opts.SourceBranch)
		}

		if opts.Description == "" {
			description, err := mrBody(commits, opts.FillCommitBody)
			if err != nil {
				return err
			}
			opts.Description = description
		}
	}
	return nil
}

func handlePush(opts *CreateOpts, remote *glrepo.Remote) error {
	if opts.ShouldPush {
		sourceRemote := remote

		sourceBranch := opts.SourceBranch

		if sourceBranch != "" {
			if idx := strings.IndexRune(sourceBranch, ':'); idx >= 0 {
				sourceBranch = sourceBranch[idx+1:]
			}
		}

		if c, err := git.UncommittedChangeCount(); c != 0 {
			if err != nil {
				return err
			}
			fmt.Fprintf(opts.IO.StdErr, "\nwarning: you have %s\n", utils.Pluralize(c, "uncommitted change"))
		}
		err := git.Push(sourceRemote.Name, fmt.Sprintf("HEAD:%s", sourceBranch), opts.IO.StdOut, opts.IO.StdErr)
		if err == nil {
			branchConfig := git.ReadBranchConfig(sourceBranch)
			if branchConfig.RemoteName == "" && (branchConfig.MergeRef == "" || branchConfig.RemoteURL == nil) {
				// No remote is set so set it
				_ = git.SetUpstream(sourceRemote.Name, sourceBranch, opts.IO.StdOut, opts.IO.StdErr)
			}
		}
		return err
	}

	return nil
}

func previewMR(opts *CreateOpts) error {
	repo, err := opts.BaseRepo()
	if err != nil {
		return err
	}

	cfg, err := opts.Config()
	if err != nil {
		return err
	}

	openURL, err := generateMRCompareURL(opts)
	if err != nil {
		return err
	}

	if opts.IO.IsOutputTTY() {
		fmt.Fprintf(opts.IO.StdErr, "Opening %s in your browser.\n", utils.DisplayURL(openURL))
	}
	browser, _ := cfg.Get(repo.RepoHost(), "browser")
	return utils.OpenInBrowser(openURL, browser)
}

func generateMRCompareURL(opts *CreateOpts) (string, error) {
	description := opts.Description

	if len(opts.Labels) > 0 {
		// this uses the slash commands to add labels to the description
		// See https://docs.gitlab.com/ee/user/project/quick_actions.html
		// See also https://gitlab.com/gitlab-org/gitlab-foss/-/issues/19731#note_32550046
		description += fmt.Sprintf("\n/label ~%s", strings.Join(opts.Labels, ", ~"))
	}
	if len(opts.Assignees) > 0 {
		// this uses the slash commands to add assignees to the description
		description += fmt.Sprintf("\n/assign %s", strings.Join(opts.Assignees, ", "))
	}
	if len(opts.Reviewers) > 0 {
		// this uses the slash commands to add reviewers to the description
		description += fmt.Sprintf("\n/reviewer %s", strings.Join(opts.Reviewers, ", "))
	}
	if opts.Milestone != 0 {
		// this uses the slash commands to add milestone to the description
		description += fmt.Sprintf("\n/milestone %%%d", opts.Milestone)
	}

	// The merge request **must** be opened against the head repo
	u, err := url.Parse(opts.SourceProject.WebURL)
	if err != nil {
		return "", err
	}

	u.Path += "/-/merge_requests/new"
	q := u.Query()
	q.Set("merge_request[title]", opts.Title)
	q.Add("merge_request[description]", description)
	q.Add("merge_request[source_branch]", opts.SourceBranch)
	q.Add("merge_request[target_branch]", opts.TargetBranch)
	q.Add("merge_request[source_project_id]", strconv.Itoa(opts.SourceProject.ID))
	q.Add("merge_request[target_project_id]", strconv.Itoa(opts.TargetProject.ID))
	u.RawQuery = q.Encode()

	return u.String(), nil
}

func ResolvedHeadRepo(f *cmdutils.Factory) func() (glrepo.Interface, error) {
	return func() (glrepo.Interface, error) {
		httpClient, err := f.HttpClient()
		if err != nil {
			return nil, err
		}
		remotes, err := f.Remotes()
		if err != nil {
			return nil, err
		}
		repoContext, err := glrepo.ResolveRemotesToRepos(remotes, httpClient, "")
		if err != nil {
			return nil, err
		}
		headRepo, err := repoContext.HeadRepo(f.IO.PromptEnabled())
		if err != nil {
			return nil, err
		}

		return headRepo, nil
	}
}

func headRepoOverride(opts *CreateOpts, repo string) error {
	opts.HeadRepo = func() (glrepo.Interface, error) {
		return glrepo.FromFullName(repo)
	}
	return nil
}

func repoRemote(opts *CreateOpts, repo glrepo.Interface, project *gitlab.Project, remoteName string) (*glrepo.Remote, error) {
	remotes, err := opts.Remotes()
	if err != nil {
		return nil, err
	}
	repoRemote, _ := remotes.FindByRepo(repo.RepoOwner(), repo.RepoName())
	if repoRemote == nil {
		cfg, err := opts.Config()
		if err != nil {
			return nil, err
		}
		gitProtocol, _ := cfg.Get(repo.RepoHost(), "git_protocol")
		repoURL := glrepo.RemoteURL(project, gitProtocol)

		gitRemote, err := git.AddRemote(remoteName, repoURL)
		if err != nil {
			return nil, fmt.Errorf("error adding remote: %w", err)
		}
		repoRemote = &glrepo.Remote{
			Remote: gitRemote,
			Repo:   repo,
		}
	}

	return repoRemote, nil
}

func getTargetBranch(baseRepoRemote *glrepo.Remote) string {
	br, _ := git.GetDefaultBranch(baseRepoRemote.Name)
	// we ignore the error since git.GetDefaultBranch returns master and an error
	// if the default branch cannot be determined
	return br
}

// createRecoverSaveFile will try save the issue create options to a file
func createRecoverSaveFile(opts *CreateOpts) error {
	glRepo, err := opts.BaseRepo()
	if err != nil {
		return err
	}

	recoverFile, err := recovery.CreateFile(glRepo.FullName(), "mr.json", opts)
	if err != nil {
		return err
	}

	fmt.Fprintf(opts.IO.StdErr, "Failed to create merge request. Created recovery file: %s\nRun the command again with the '--recover' option to retry.\n", recoverFile)
	return nil
}
