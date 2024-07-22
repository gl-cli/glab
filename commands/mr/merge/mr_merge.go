package merge

import (
	"errors"
	"fmt"
	"net/http"
	"time"

	"gitlab.com/gitlab-org/cli/pkg/surveyext"

	"github.com/AlecAivazis/survey/v2"
	"github.com/MakeNowJust/heredoc/v2"
	"github.com/avast/retry-go/v4"
	"gitlab.com/gitlab-org/cli/api"
	"gitlab.com/gitlab-org/cli/commands/mr/mrutils"
	"gitlab.com/gitlab-org/cli/pkg/prompt"

	"github.com/spf13/cobra"
	"github.com/xanzy/go-gitlab"
	"gitlab.com/gitlab-org/cli/commands/cmdutils"
)

type MRMergeMethod int

const (
	MRMergeMethodMerge MRMergeMethod = iota
	MRMergeMethodSquash
	MRMergeMethodRebase
)

type MergeOpts struct {
	SetAutoMerge       bool
	SquashBeforeMerge  bool
	RebaseBeforeMerge  bool
	RemoveSourceBranch bool
	SkipPrompts        bool

	SquashMessage      string
	MergeCommitMessage string
	SHA                string

	MergeMethod MRMergeMethod
}

func NewCmdMerge(f *cmdutils.Factory) *cobra.Command {
	opts := &MergeOpts{
		MergeMethod: MRMergeMethodMerge,
	}

	mrMergeCmd := &cobra.Command{
		Use:     "merge {<id> | <branch>}",
		Short:   `Merge or accept a merge request.`,
		Long:    ``,
		Aliases: []string{"accept"},
		Example: heredoc.Doc(`
			$ glab mr merge 235
			$ glab mr accept 235

			# Finds open merge request from current branch
			$ glab mr merge
		`),
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			var err error
			c := f.IO.Color()

			if opts.SquashBeforeMerge && opts.RebaseBeforeMerge {
				return &cmdutils.FlagError{Err: errors.New("only one of --rebase or --squash can be enabled")}
			}

			if !opts.SquashBeforeMerge && opts.SquashMessage != "" {
				return &cmdutils.FlagError{Err: errors.New("--squash-message can only be used with --squash.")}
			}

			apiClient, err := f.HttpClient()
			if err != nil {
				return err
			}

			mr, repo, err := mrutils.MRFromArgs(f, args, "opened")
			if err != nil {
				return err
			}

			if err = mrutils.MRCheckErrors(mr, mrutils.MRCheckErrOptions{
				WorkInProgress: true,
				Closed:         true,
				Merged:         true,
				Conflict:       true,
				PipelineStatus: true,
				MergePrivilege: true,
			}); err != nil {
				return err
			}

			if !cmd.Flags().Changed("when-pipeline-succeeds") &&
				!cmd.Flags().Changed("auto-merge") &&
				f.IO.IsOutputTTY() &&
				mr.Pipeline != nil &&
				f.IO.PromptEnabled() &&
				!opts.SkipPrompts {
				_ = prompt.Confirm(&opts.SetAutoMerge, "Set auto-merge?", true)
			}

			if f.IO.IsOutputTTY() && !opts.SkipPrompts {
				if !opts.SquashBeforeMerge && !opts.RebaseBeforeMerge && opts.MergeCommitMessage == "" {
					opts.MergeMethod, err = mergeMethodSurvey()
					if err != nil {
						return err
					}
					if opts.MergeMethod == MRMergeMethodSquash {
						opts.SquashBeforeMerge = true
					} else if opts.MergeMethod == MRMergeMethodRebase {
						opts.RebaseBeforeMerge = true
					}
				}

				if opts.MergeCommitMessage == "" && opts.SquashMessage == "" {
					action, err := confirmSurvey(opts.MergeMethod != MRMergeMethodRebase)
					if err != nil {
						return fmt.Errorf("unable to prompt: %w", err)
					}

					if action == cmdutils.EditCommitMessageAction {
						var mergeMessage string

						editor, err := cmdutils.GetEditor(f.Config)
						if err != nil {
							return err
						}
						mergeMessage, err = surveyext.Edit(editor, "*.md", mr.Title, f.IO.In, f.IO.StdOut, f.IO.StdErr, nil)
						if err != nil {
							return err
						}

						if opts.SquashBeforeMerge {
							opts.SquashMessage = mergeMessage
						} else {
							opts.MergeCommitMessage = mergeMessage
						}

						action, err = confirmSurvey(false)
						if err != nil {
							return fmt.Errorf("unable to confirm: %w", err)
						}
					}
					if action == cmdutils.CancelAction {
						fmt.Fprintln(f.IO.StdErr, "Cancelled.")
						return cmdutils.SilentError
					}
				}
			}

			mergeOpts := &gitlab.AcceptMergeRequestOptions{}
			if opts.MergeCommitMessage != "" {
				mergeOpts.MergeCommitMessage = gitlab.Ptr(opts.MergeCommitMessage)
			}
			if opts.SquashMessage != "" {
				mergeOpts.SquashCommitMessage = gitlab.Ptr(opts.SquashMessage)
			}
			if opts.SquashBeforeMerge {
				mergeOpts.Squash = gitlab.Ptr(true)
			}
			if opts.RemoveSourceBranch {
				mergeOpts.ShouldRemoveSourceBranch = gitlab.Ptr(true)
			}
			if opts.SetAutoMerge && mr.Pipeline != nil {
				if mr.Pipeline.Status == "canceled" || mr.Pipeline.Status == "failed" {
					fmt.Fprintln(f.IO.StdOut, c.FailedIcon(), "Pipeline status:", mr.Pipeline.Status)
					fmt.Fprintln(f.IO.StdOut, c.FailedIcon(), "Cannot perform merge action")
					return cmdutils.SilentError
				}
				mergeOpts.MergeWhenPipelineSucceeds = gitlab.Ptr(true)
			}
			if opts.SHA != "" {
				mergeOpts.SHA = gitlab.Ptr(opts.SHA)
			}

			if opts.RebaseBeforeMerge {
				err := mrutils.RebaseMR(f.IO, apiClient, repo, mr, nil)
				if err != nil {
					return err
				}
			}

			f.IO.StartSpinner("Merging merge request !%d.", mr.IID)

			// Store the IID of the merge request here before overriding the `mr` variable
			// inside the retry function, if the function fails at first the `mr` is replaced
			// with `nil` and will cause a crash on the second run
			mrIID := mr.IID

			err = retry.Do(func() error {
				var resp *gitlab.Response
				mr, resp, err = api.MergeMR(apiClient, repo.FullName(), mrIID, mergeOpts)
				if err != nil {
					// https://docs.gitlab.com/ee/api/merge_requests.html#accept-mr
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
			f.IO.StopSpinner("")
			isMerged := true
			if opts.SetAutoMerge {
				if mr.Pipeline == nil {
					fmt.Fprintln(f.IO.StdOut, c.WarnIcon(), "No pipeline running on", mr.SourceBranch)
				} else {
					switch mr.Pipeline.Status {
					case "success":
						fmt.Fprintln(f.IO.StdOut, c.GreenCheck(), "Pipeline succeeded.")
					default:
						fmt.Fprintln(f.IO.StdOut, c.WarnIcon(), "Pipeline status:", mr.Pipeline.Status)
						if mr.State != "merged" {
							fmt.Fprintln(f.IO.StdOut, c.GreenCheck(), "Will auto-merge")
							isMerged = false
						}
					}
				}
			}
			if isMerged {
				action := "Merged!"
				switch opts.MergeMethod {
				case MRMergeMethodRebase:
					action = "Rebased and merged!"
				case MRMergeMethodSquash:
					action = "Squashed and merged!"
				}
				fmt.Fprintln(f.IO.StdOut, c.GreenCheck(), action)
			}
			fmt.Fprintln(f.IO.StdOut, mrutils.DisplayMR(c, mr, f.IO.IsaTTY))
			return nil
		},
	}

	mrMergeCmd.Flags().StringVarP(&opts.SHA, "sha", "", "", "Merge commit SHA.")
	mrMergeCmd.Flags().BoolVarP(&opts.RemoveSourceBranch, "remove-source-branch", "d", false, "Remove source branch on merge.")
	mrMergeCmd.Flags().BoolVarP(&opts.SetAutoMerge, "auto-merge", "", true, "Set auto-merge.")
	mrMergeCmd.Flags().StringVarP(&opts.MergeCommitMessage, "message", "m", "", "Custom merge commit message.")
	mrMergeCmd.Flags().StringVarP(&opts.SquashMessage, "squash-message", "", "", "Custom squash commit message.")
	mrMergeCmd.Flags().BoolVarP(&opts.SquashBeforeMerge, "squash", "s", false, "Squash commits on merge.")
	mrMergeCmd.Flags().BoolVarP(&opts.RebaseBeforeMerge, "rebase", "r", false, "Rebase the commits onto the base branch.")
	mrMergeCmd.Flags().BoolVarP(&opts.SkipPrompts, "yes", "y", false, "Skip submission confirmation prompt.")

	mrMergeCmd.Flags().BoolVarP(&opts.SetAutoMerge, "when-pipeline-succeeds", "", true, "Merge only when pipeline succeeds")
	_ = mrMergeCmd.Flags().MarkDeprecated("when-pipeline-succeeds", "use --auto-merge instead.")

	return mrMergeCmd
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
