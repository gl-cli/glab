package update

import (
	"errors"
	"fmt"
	"strings"

	"gitlab.com/gitlab-org/cli/api"
	"gitlab.com/gitlab-org/cli/commands/cmdutils"
	"gitlab.com/gitlab-org/cli/commands/issue/issueutils"

	"github.com/MakeNowJust/heredoc/v2"
	"github.com/spf13/cobra"
	"github.com/xanzy/go-gitlab"
)

func NewCmdUpdate(f *cmdutils.Factory) *cobra.Command {
	issueUpdateCmd := &cobra.Command{
		Use:   "update <id>",
		Short: `Update issue`,
		Long:  ``,
		Example: heredoc.Doc(`
	glab issue update 42 --label ui,ux
	glab issue update 42 --unlabel working
	`),
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			var err error
			var actions []string
			var ua *cmdutils.UserAssignments
			out := f.IO.StdOut
			c := f.IO.Color()

			if cmd.Flags().Changed("unassign") && cmd.Flags().Changed("assignee") {
				return &cmdutils.FlagError{Err: fmt.Errorf("--assignee and --unassign are mutually exclusive.")}
			}

			// Parse assignees early so we can fail early in case of conflicts
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

			if cmd.Flags().Changed("lock-discussion") && cmd.Flags().Changed("unlock-discussion") {
				return &cmdutils.FlagError{
					Err: errors.New("--lock-discussion and --unlock-discussion can't be used together."),
				}
			}
			if cmd.Flags().Changed("confidential") && cmd.Flags().Changed("public") {
				return &cmdutils.FlagError{Err: errors.New("--public and --confidential can't be used together.")}
			}

			apiClient, err := f.HttpClient()
			if err != nil {
				return err
			}
			issue, repo, err := issueutils.IssueFromArg(apiClient, f.BaseRepo, args[0])
			if err != nil {
				return err
			}
			l := &gitlab.UpdateIssueOptions{}

			if m, _ := cmd.Flags().GetString("title"); m != "" {
				actions = append(actions, fmt.Sprintf("updated title to %q", m))
				l.Title = gitlab.Ptr(m)
			}
			if m, _ := cmd.Flags().GetBool("lock-discussion"); m {
				actions = append(actions, "locked discussion")
				l.DiscussionLocked = gitlab.Ptr(m)
			}
			if m, _ := cmd.Flags().GetBool("unlock-discussion"); m {
				actions = append(actions, "unlocked dicussion")
				l.DiscussionLocked = gitlab.Ptr(false)
			}

			if m, _ := cmd.Flags().GetString("description"); m != "" {
				actions = append(actions, "updated description")

				// Edit the description via editor
				if m == "-" {

					// Fetch the current issue and description
					apiClient, err := f.HttpClient()
					if err != nil {
						return err
					}
					issue, _, err := issueutils.IssueFromArg(apiClient, f.BaseRepo, args[0])
					if err != nil {
						return err
					}

					editor, err := cmdutils.GetEditor(f.Config)
					if err != nil {
						return err
					}

					l.Description = gitlab.Ptr("")
					err = cmdutils.EditorPrompt(l.Description, "Description", issue.Description, editor)
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
			if m, _ := cmd.Flags().GetBool("public"); m {
				actions = append(actions, "made public")
				l.Confidential = gitlab.Ptr(false)
			}
			if m, _ := cmd.Flags().GetBool("confidential"); m {
				actions = append(actions, "made confidential")
				l.Confidential = gitlab.Ptr(true)
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
					issue, err := api.GetIssue(apiClient, repo.FullName(), issue.IID)
					if err != nil {
						return err
					}
					l.AssigneeIDs, actions, err = ua.UsersFromAddRemove(issue.Assignees, nil, apiClient, actions)
					if err != nil {
						return err
					}
				}
			}

			fmt.Fprintf(out, "- Updating issue #%d\n", issue.IID)

			issue, err = api.UpdateIssue(apiClient, repo.FullName(), issue.IID, l)
			if err != nil {
				return err
			}

			for _, s := range actions {
				fmt.Fprintln(out, c.GreenCheck(), s)
			}

			fmt.Fprintln(out, issueutils.DisplayIssue(c, issue, f.IO.IsaTTY))
			return nil
		},
	}

	issueUpdateCmd.Flags().StringP("title", "t", "", "Title of issue.")
	issueUpdateCmd.Flags().BoolP("lock-discussion", "", false, "Lock discussion on issue.")
	issueUpdateCmd.Flags().BoolP("unlock-discussion", "", false, "Unlock discussion on issue.")
	issueUpdateCmd.Flags().StringP("description", "d", "", "Issue description. Set to \"-\" to open an editor.")
	issueUpdateCmd.Flags().StringSliceP("label", "l", []string{}, "Add labels.")
	issueUpdateCmd.Flags().StringSliceP("unlabel", "u", []string{}, "Remove labels.")
	issueUpdateCmd.Flags().BoolP("public", "p", false, "Make issue public.")
	issueUpdateCmd.Flags().BoolP("confidential", "c", false, "Make issue confidential")
	issueUpdateCmd.Flags().StringP("milestone", "m", "", "Title of the milestone to assign Set to \"\" or 0 to unassign.")
	issueUpdateCmd.Flags().
		StringSliceP("assignee", "a", []string{}, "Assign users by username. Prefix with '!' or '-' to remove from existing assignees, or '+' to add new. Otherwise, replace existing assignees with these users.")
	issueUpdateCmd.Flags().Bool("unassign", false, "Unassign all users.")

	return issueUpdateCmd
}
