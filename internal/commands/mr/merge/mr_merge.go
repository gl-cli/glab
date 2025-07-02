package merge

import (
	"errors"
	"fmt"
	"net/http"
	"time"

	"gitlab.com/gitlab-org/cli/internal/config"
	"gitlab.com/gitlab-org/cli/internal/dbg"
	"gitlab.com/gitlab-org/cli/internal/iostreams"
	"gitlab.com/gitlab-org/cli/internal/surveyext"

	"github.com/AlecAivazis/survey/v2"
	"github.com/MakeNowJust/heredoc/v2"
	"github.com/avast/retry-go/v4"
	"gitlab.com/gitlab-org/cli/internal/commands/mr/mrutils"
	"gitlab.com/gitlab-org/cli/internal/prompt"

	"github.com/spf13/cobra"
	gitlab "gitlab.com/gitlab-org/api/client-go"
	"gitlab.com/gitlab-org/cli/internal/cmdutils"
)

type MRMergeMethod int

const (
	MRMergeMethodMerge MRMergeMethod = iota
	MRMergeMethodSquash
	MRMergeMethodRebase
)

type options struct {
	io         *iostreams.IOStreams
	httpClient func() (*gitlab.Client, error)
	config     func() config.Config

	setAutoMerge       bool
	squashBeforeMerge  bool
	rebaseBeforeMerge  bool
	removeSourceBranch bool
	skipPrompts        bool

	squashMessage      string
	mergeCommitMessage string
	sha                string

	mergeMethod MRMergeMethod
}

func NewCmdMerge(f cmdutils.Factory) *cobra.Command {
	opts := &options{
		io:         f.IO(),
		httpClient: f.HttpClient,
		config:     f.Config,

		mergeMethod: MRMergeMethodMerge,
	}

	mrMergeCmd := &cobra.Command{
		Use:     "merge {<id> | <branch>}",
		Short:   `Merge or accept a merge request.`,
		Long:    ``,
		Aliases: []string{"accept"},
		Example: heredoc.Doc(`
			# Merge a merge request
			$ glab mr merge 235
			$ glab mr accept 235

			# Finds open merge request from current branch
			$ glab mr merge
		`),
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := opts.validate(); err != nil {
				return err
			}

			return opts.run(f, cmd, args)
		},
	}

	mrMergeCmd.Flags().StringVarP(&opts.sha, "sha", "", "", "Merge commit SHA.")
	mrMergeCmd.Flags().BoolVarP(&opts.removeSourceBranch, "remove-source-branch", "d", false, "Remove source branch on merge.")
	mrMergeCmd.Flags().BoolVarP(&opts.setAutoMerge, "auto-merge", "", true, "Set auto-merge.")
	mrMergeCmd.Flags().StringVarP(&opts.mergeCommitMessage, "message", "m", "", "Custom merge commit message.")
	mrMergeCmd.Flags().StringVarP(&opts.squashMessage, "squash-message", "", "", "Custom squash commit message.")
	mrMergeCmd.Flags().BoolVarP(&opts.squashBeforeMerge, "squash", "s", false, "Squash commits on merge.")
	mrMergeCmd.Flags().BoolVarP(&opts.rebaseBeforeMerge, "rebase", "r", false, "Rebase the commits onto the base branch.")
	mrMergeCmd.Flags().BoolVarP(&opts.skipPrompts, "yes", "y", false, "Skip submission confirmation prompt.")

	mrMergeCmd.Flags().BoolVarP(&opts.setAutoMerge, "when-pipeline-succeeds", "", true, "Merge only when pipeline succeeds")
	_ = mrMergeCmd.Flags().MarkDeprecated("when-pipeline-succeeds", "use --auto-merge instead.")
	mrMergeCmd.MarkFlagsMutuallyExclusive("squash", "rebase")

	return mrMergeCmd
}

func (o *options) validate() error {
	if !o.squashBeforeMerge && o.squashMessage != "" {
		return &cmdutils.FlagError{Err: errors.New("--squash-message can only be used with --squash.")}
	}

	return nil
}

func (o *options) run(x cmdutils.Factory, cmd *cobra.Command, args []string) error {
	c := o.io.Color()

	apiClient, err := o.httpClient()
	if err != nil {
		return err
	}

	mr, repo, err := mrutils.MRFromArgs(x, args, "opened")
	if err != nil {
		return err
	}

	if err = mrutils.MRCheckErrors(mr, mrutils.MRCheckErrOptions{
		Draft:          true,
		Closed:         true,
		Merged:         true,
		Conflict:       true,
		PipelineStatus: true,
		MergePrivilege: true,
	}); err != nil {
		dbg.Debug("MRCheckErrors failed")
		return err
	}

	if !cmd.Flags().Changed("when-pipeline-succeeds") &&
		!cmd.Flags().Changed("auto-merge") &&
		o.io.IsOutputTTY() &&
		mr.Pipeline != nil &&
		o.io.PromptEnabled() &&
		!o.skipPrompts {
		_ = prompt.Confirm(&o.setAutoMerge, "Set auto-merge?", true)
	}

	if o.io.IsOutputTTY() && !o.skipPrompts {
		if !o.squashBeforeMerge && !o.rebaseBeforeMerge && o.mergeCommitMessage == "" {
			o.mergeMethod, err = mergeMethodSurvey()
			if err != nil {
				return err
			}
			if o.mergeMethod == MRMergeMethodSquash {
				o.squashBeforeMerge = true
			} else if o.mergeMethod == MRMergeMethodRebase {
				o.rebaseBeforeMerge = true
			}
		}

		if o.mergeCommitMessage == "" && o.squashMessage == "" {
			action, err := confirmSurvey(o.mergeMethod != MRMergeMethodRebase)
			if err != nil {
				return fmt.Errorf("unable to prompt: %w", err)
			}

			if action == cmdutils.EditCommitMessageAction {
				var mergeMessage string

				editor, err := cmdutils.GetEditor(o.config)
				if err != nil {
					return err
				}
				mergeMessage, err = surveyext.Edit(editor, "*.md", mr.Title, o.io.In, o.io.StdOut, o.io.StdErr, nil)
				if err != nil {
					return err
				}

				if o.squashBeforeMerge {
					o.squashMessage = mergeMessage
				} else {
					o.mergeCommitMessage = mergeMessage
				}

				action, err = confirmSurvey(false)
				if err != nil {
					return fmt.Errorf("unable to confirm: %w", err)
				}
			}
			if action == cmdutils.CancelAction {
				fmt.Fprintln(o.io.StdErr, "Cancelled.")
				return cmdutils.SilentError
			}
		}
	}

	mergeOpts := &gitlab.AcceptMergeRequestOptions{}
	if o.mergeCommitMessage != "" {
		mergeOpts.MergeCommitMessage = gitlab.Ptr(o.mergeCommitMessage)
	}
	if o.squashMessage != "" {
		mergeOpts.SquashCommitMessage = gitlab.Ptr(o.squashMessage)
	}
	if o.squashBeforeMerge {
		mergeOpts.Squash = gitlab.Ptr(true)
	}
	if o.removeSourceBranch {
		mergeOpts.ShouldRemoveSourceBranch = gitlab.Ptr(true)
	}
	if o.setAutoMerge && mr.Pipeline != nil {
		if mr.Pipeline.Status == "canceled" || mr.Pipeline.Status == "failed" {
			fmt.Fprintln(o.io.StdOut, c.FailedIcon(), "Pipeline status:", mr.Pipeline.Status)
			fmt.Fprintln(o.io.StdOut, c.FailedIcon(), "Cannot perform merge action")
			return cmdutils.SilentError
		}
		mergeOpts.MergeWhenPipelineSucceeds = gitlab.Ptr(true)
	}
	if o.sha != "" {
		mergeOpts.SHA = gitlab.Ptr(o.sha)
	}

	if o.rebaseBeforeMerge {
		err := mrutils.RebaseMR(o.io, apiClient, repo, mr, nil)
		if err != nil {
			return err
		}
	}

	o.io.StartSpinner("Merging merge request !%d.", mr.IID)

	// Store the IID of the merge request here before overriding the `mr` variable
	// inside the retry function, if the function fails at first the `mr` is replaced
	// with `nil` and will cause a crash on the second run
	mrIID := mr.IID

	err = retry.Do(func() error {
		var resp *gitlab.Response
		mr, resp, err = apiClient.MergeRequests.AcceptMergeRequest(repo.FullName(), mrIID, mergeOpts)
		if err != nil {
			// https://docs.gitlab.com/api/merge_requests/#merge-a-merge-request
			// `406` is the documented status code we will receive if the
			// branch cannot be merged, this will catch situations where
			// there are actually conflicts in the branch instead of just
			// the situation we want to workaround (GitLab thinking branch
			// is not mergeable right after a rebase), but we want to catch
			// situations where the user rebased via external sources like
			// the WebUI or running `glab rebase` before trying to merge
			if resp.StatusCode == http.StatusNotAcceptable {
				return err
			}

			// Return an unrecoverable error if we are not rebasing OR if the
			// error isn't the one we are working around, this makes the retry
			// to quit instead of trying again
			return retry.Unrecoverable(err)
		}
		return err
	}, retry.Attempts(3), retry.Delay(time.Second*6))
	if err != nil {
		return err
	}
	o.io.StopSpinner("")
	isMerged := true
	if o.setAutoMerge {
		if mr.Pipeline == nil {
			fmt.Fprintln(o.io.StdOut, c.WarnIcon(), "No pipeline running on", mr.SourceBranch)
		} else {
			switch mr.Pipeline.Status {
			case "success":
				fmt.Fprintln(o.io.StdOut, c.GreenCheck(), "Pipeline succeeded.")
			default:
				fmt.Fprintln(o.io.StdOut, c.WarnIcon(), "Pipeline status:", mr.Pipeline.Status)
				if mr.State != "merged" {
					fmt.Fprintln(o.io.StdOut, c.GreenCheck(), "Will auto-merge")
					isMerged = false
				}
			}
		}
	}
	if isMerged {
		action := "Merged!"
		switch o.mergeMethod {
		case MRMergeMethodRebase:
			action = "Rebased and merged!"
		case MRMergeMethodSquash:
			action = "Squashed and merged!"
		}
		fmt.Fprintln(o.io.StdOut, c.GreenCheck(), action)
	}
	fmt.Fprintln(o.io.StdOut, mrutils.DisplayMR(c, &mr.BasicMergeRequest, o.io.IsaTTY))
	return nil
}

func mergeMethodSurvey() (MRMergeMethod, error) {
	type mergeOption struct {
		title  string
		method MRMergeMethod
	}

	mergeOpts := []mergeOption{
		{title: "Create a merge commit", method: MRMergeMethodMerge},
		{title: "Rebase and merge", method: MRMergeMethodRebase},
		{title: "Squash and merge", method: MRMergeMethodSquash},
	}

	var surveyOpts []string
	for _, v := range mergeOpts {
		surveyOpts = append(surveyOpts, v.title)
	}

	mergeQuestion := &survey.Select{
		Message: "What merge method do you want to use?",
		Options: surveyOpts,
	}

	var result int
	err := prompt.AskOne(mergeQuestion, &result)
	return mergeOpts[result].method, err
}

func confirmSurvey(allowEditMsg bool) (cmdutils.Action, error) {
	const (
		submitLabel        = "Submit"
		editCommitMsgLabel = "Edit commit message"
		cancelLabel        = "Cancel"
	)

	options := []string{submitLabel}
	if allowEditMsg {
		options = append(options, editCommitMsgLabel)
	}
	options = append(options, cancelLabel)

	var result string
	submit := &survey.Select{
		Message: "What's next?",
		Options: options,
	}
	err := prompt.AskOne(submit, &result)
	if err != nil {
		return cmdutils.CancelAction, fmt.Errorf("could not prompt: %w", err)
	}

	switch result {
	case submitLabel:
		return cmdutils.SubmitAction, nil
	case editCommitMsgLabel:
		return cmdutils.EditCommitMessageAction, nil
	default:
		return cmdutils.CancelAction, nil
	}
}
