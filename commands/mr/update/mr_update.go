package update

import (
	"errors"
	"fmt"
	"strings"

	"gitlab.com/gitlab-org/cli/api"
	"gitlab.com/gitlab-org/cli/commands/cmdutils"
	"gitlab.com/gitlab-org/cli/commands/mr/mrutils"

	"github.com/MakeNowJust/heredoc/v2"
	"github.com/spf13/cobra"
	"github.com/xanzy/go-gitlab"
)

func NewCmdUpdate(f *cmdutils.Factory) *cobra.Command {
	mrUpdateCmd := &cobra.Command{
		Use:   "update [<id> | <branch>]",
		Short: `Update a merge request.`,
		Long:  ``,
		Example: heredoc.Doc(`
	$ glab mr update 23 --ready
	$ glab mr update 23 --draft

	# Updates the merge request for the current branch
	$ glab mr update --draft
	`),
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			var err error
			var actions []string
			var ua *cmdutils.UserAssignments // assignees
			var ur *cmdutils.UserAssignments // reviewers
			c := f.IO.Color()

			if cmd.Flags().Changed("unassign") && cmd.Flags().Changed("assignee") {
				return &cmdutils.FlagError{Err: fmt.Errorf("--assignee and --unassign are mutually exclusive.")}
			}

			// Parse assignees Early so we can fail early in case of conflicts
			if cmd.Flags().Changed("assignee") {
				givenAssignees, err := cmd.Flags().GetStringSlice("assignee")
				if err != nil {
					return err
				}
				ua = cmdutils.ParseAssignees(givenAssignees)

				err = ua.VerifyAssignees()
				if err != nil {
					return &cmdutils.FlagError{Err: fmt.Errorf("--assignee: %w", err)}
				}
			}

			if cmd.Flags().Changed("reviewer") {
				givenReviewers, err := cmd.Flags().GetStringSlice("reviewer")
				if err != nil {
					return err
				}
				ur = cmdutils.ParseAssignees(givenReviewers)
				ur.AssignmentType = cmdutils.ReviewerAssignment

				err = ur.VerifyAssignees()
				if err != nil {
					return &cmdutils.FlagError{Err: fmt.Errorf("--reviewer: %w", err)}
				}
			}

			if cmd.Flags().Changed("lock-discussion") && cmd.Flags().Changed("unlock-discussion") {
				return &cmdutils.FlagError{
					Err: errors.New("--lock-discussion and --unlock-discussion can't be used together."),
				}
			}

			apiClient, err := f.HttpClient()
			if err != nil {
				return err
			}

			mr, repo, err := mrutils.MRFromArgs(f, args, "any")
			if err != nil {
				return err
			}

			l := &gitlab.UpdateMergeRequestOptions{}
			var mergeTitle string

			isDraft, _ := cmd.Flags().GetBool("draft")
			isWIP, _ := cmd.Flags().GetBool("wip")
			if m, _ := cmd.Flags().GetString("title"); m != "" {
				actions = append(actions, fmt.Sprintf("updated title to %q", m))
				mergeTitle = m
			}
			if mergeTitle == "" {
				opts := &gitlab.GetMergeRequestsOptions{}
				mr, err := api.GetMR(apiClient, repo.FullName(), mr.IID, opts)
				if err != nil {
					return err
				}
				mergeTitle = mr.Title
			}
			if isDraft || isWIP {
				if isDraft {
					actions = append(actions, "marked as Draft")
					mergeTitle = "Draft: " + mergeTitle
				} else {
					actions = append(actions, "marked as WIP")
					mergeTitle = "WIP: " + mergeTitle
				}
			} else if isReady, _ := cmd.Flags().GetBool("ready"); isReady {
				actions = append(actions, "marked as ready")
				mergeTitle = strings.TrimPrefix(mergeTitle, "Draft:")
				mergeTitle = strings.TrimPrefix(mergeTitle, "draft:")
				mergeTitle = strings.TrimPrefix(mergeTitle, "DRAFT:")
				mergeTitle = strings.TrimPrefix(mergeTitle, "WIP:")
				mergeTitle = strings.TrimPrefix(mergeTitle, "wip:")
				mergeTitle = strings.TrimPrefix(mergeTitle, "Wip:")
				mergeTitle = strings.TrimSpace(mergeTitle)
			}

			l.Title = gitlab.Ptr(mergeTitle)
			if m, _ := cmd.Flags().GetBool("lock-discussion"); m {
				actions = append(actions, "locked discussion")
				l.DiscussionLocked = gitlab.Ptr(m)
			}
			if m, _ := cmd.Flags().GetBool("unlock-discussion"); m {
				actions = append(actions, "unlocked discussion")
				l.DiscussionLocked = gitlab.Ptr(false)
			}

			if m, _ := cmd.Flags().GetString("description"); m != "" {
				actions = append(actions, "updated description")

				// Edit the description via editor
				if m == "-" {

					mr, _, err := mrutils.MRFromArgs(f, args, "any")
					if err != nil {
						return err
					}

					editor, err := cmdutils.GetEditor(f.Config)
					if err != nil {
						return err
					}

					l.Description = gitlab.Ptr("")
					err = cmdutils.EditorPrompt(l.Description, "Description", mr.Description, editor)
					if err != nil {
						return err
					}
				} else {
					l.Description = gitlab.Ptr(m)
				}
			}

			if m, _ := cmd.Flags().GetStringSlice("label"); len(m) != 0 {
				actions = append(actions, fmt.Sprintf("added labels %s", strings.Join(m, " ")))
				l.AddLabels = (*gitlab.LabelOptions)(&m)
			}
			if m, _ := cmd.Flags().GetStringSlice("unlabel"); len(m) != 0 {
				actions = append(actions, fmt.Sprintf("removed labels %s", strings.Join(m, " ")))
				l.RemoveLabels = (*gitlab.LabelOptions)(&m)
			}
			if m, _ := cmd.Flags().GetString("target-branch"); m != "" {
				actions = append(actions, fmt.Sprintf("set target branch to %q", m))
				l.TargetBranch = gitlab.Ptr(m)
			}
			if ok := cmd.Flags().Changed("milestone"); ok {
				if m, _ := cmd.Flags().GetString("milestone"); m != "" || m == "0" {
					mID, err := cmdutils.ParseMilestone(apiClient, repo, m)
					if err != nil {
						return err
					}
					actions = append(actions, fmt.Sprintf("added milestone %q", m))
					l.MilestoneID = gitlab.Ptr(mID)
				} else {
					// Unassign the Milestone
					actions = append(actions, "unassigned milestone")
					l.MilestoneID = gitlab.Ptr(0)
				}
			}
			if cmd.Flags().Changed("unassign") {
				l.AssigneeIDs = &[]int{0} // 0 or an empty int[] is the documented way to unassign
				actions = append(actions, "unassigned all users")
			}
			if ua != nil {
				if len(ua.ToReplace) != 0 {
					l.AssigneeIDs, actions, err = ua.UsersFromReplaces(apiClient, actions)
					if err != nil {
						return err
					}
				} else if len(ua.ToAdd) != 0 || len(ua.ToRemove) != 0 {
					l.AssigneeIDs, actions, err = ua.UsersFromAddRemove(nil, mr.Assignees, apiClient, actions)
					if err != nil {
						return err
					}
				}
			}

			if ur != nil {
				if len(ur.ToReplace) != 0 {
					l.ReviewerIDs, actions, err = ur.UsersFromReplaces(apiClient, actions)
					if err != nil {
						return err
					}
				} else if len(ur.ToAdd) != 0 || len(ur.ToRemove) != 0 {
					l.ReviewerIDs, actions, err = ur.UsersFromAddRemove(nil, mr.Reviewers, apiClient, actions)
					if err != nil {
						return err
					}
				}
			}

			if removeSource, _ := cmd.Flags().GetBool("remove-source-branch"); removeSource {

				if mr.ForceRemoveSourceBranch {
					actions = append(actions, "disabled removal of source branch on merge.")
				} else {
					actions = append(actions, "enabled removal of source branch on merge.")
				}

				l.RemoveSourceBranch = gitlab.Ptr(!mr.ForceRemoveSourceBranch)
			}

			if squashBeforeMerge, _ := cmd.Flags().GetBool("squash-before-merge"); squashBeforeMerge {

				if mr.Squash {
					actions = append(actions, "disabled squashing of commits before merge.")
				} else {
					actions = append(actions, "enabled squashing of commits before merge.")
				}

				l.Squash = gitlab.Ptr(!mr.Squash)
			}

			fmt.Fprintf(f.IO.StdOut, "- Updating merge request !%d\n", mr.IID)

			mr, err = api.UpdateMR(apiClient, repo.FullName(), mr.IID, l)
			if err != nil {
				return err
			}

			for _, s := range actions {
				fmt.Fprintln(f.IO.StdOut, c.GreenCheck(), s)
			}

			fmt.Fprintln(f.IO.StdOut, mrutils.DisplayMR(c, mr, f.IO.IsaTTY))
			return nil
		},
	}

	mrUpdateCmd.Flags().BoolP("draft", "", false, "Mark merge request as a draft.")
	mrUpdateCmd.Flags().BoolP("ready", "r", false, "Mark merge request as ready to be reviewed and merged.")
	mrUpdateCmd.Flags().BoolP("wip", "", false, "Mark merge request as a work in progress. Alternative to --draft.")
	mrUpdateCmd.Flags().StringP("title", "t", "", "Title of merge request.")
	mrUpdateCmd.Flags().BoolP("lock-discussion", "", false, "Lock discussion on merge request.")
	mrUpdateCmd.Flags().BoolP("unlock-discussion", "", false, "Unlock discussion on merge request.")
	mrUpdateCmd.Flags().StringP("description", "d", "", "Merge request description. Set to \"-\" to open an editor.")
	mrUpdateCmd.Flags().StringSliceP("label", "l", []string{}, "Add labels.")
	mrUpdateCmd.Flags().StringSliceP("unlabel", "u", []string{}, "Remove labels.")
	mrUpdateCmd.Flags().
		StringSliceP("assignee", "a", []string{}, "Assign users via username. Prefix with '!' or '-' to remove from existing assignees, '+' to add. Otherwise, replace existing assignees with given users.")
	mrUpdateCmd.Flags().
		StringSliceP("reviewer", "", []string{}, "Request review from users by their usernames. Prefix with '!' or '-' to remove from existing reviewers, '+' to add. Otherwise, replace existing reviewers with given users.")
	mrUpdateCmd.Flags().Bool("unassign", false, "Unassign all users.")
	mrUpdateCmd.Flags().
		BoolP("squash-before-merge", "", false, "Toggles the option to squash commits into a single commit when merging.")
	mrUpdateCmd.Flags().BoolP("remove-source-branch", "", false, "Toggles the removal of the source branch on merge.")
	mrUpdateCmd.Flags().StringP("milestone", "m", "", "Title of the milestone to assign. Set to \"\" or 0 to unassign.")
	mrUpdateCmd.Flags().String("target-branch", "", "Set target branch.")

	return mrUpdateCmd
}
