package edit

import (
	"strconv"

	"github.com/MakeNowJust/heredoc/v2"
	"github.com/spf13/cobra"
	gitlab "gitlab.com/gitlab-org/api/client-go"
	"gitlab.com/gitlab-org/cli/internal/api"
	"gitlab.com/gitlab-org/cli/internal/cmdutils"
	"gitlab.com/gitlab-org/cli/internal/glrepo"
	"gitlab.com/gitlab-org/cli/internal/iostreams"
	"gitlab.com/gitlab-org/cli/internal/mcpannotations"
)

type options struct {
	apiClient func(repoHost string) (*api.Client, error)
	io        *iostreams.IOStreams
	baseRepo  func() (glrepo.Interface, error)

	projectID   string
	groupID     string
	milestoneID int

	title       string
	description string
	dueDate     string
	startDate   string
	state       string
}

func NewCmdEdit(f cmdutils.Factory) *cobra.Command {
	opts := &options{
		io:        f.IO(),
		apiClient: f.ApiClient,
		baseRepo:  f.BaseRepo,
	}

	cmd := &cobra.Command{
		Use:   "edit",
		Short: "Edit a group or project milestone.",
		Long:  "",
		Example: heredoc.Doc(`
		  # Edit milestone for the current project
			$ glab milestone edit 123 --title='Example title' --due-date='2025-12-16'

			# Edit milestone for the specified project
			$ glab milestone edit 123 --title='Example group milestone' --due-date='2025-12-16' --project example-path/project-path

			# Edit milestone for the specified group
			$ glab milestone edit 123 --title='Example group milestone' --due-date='2025-12-16' --group 789
		`),
		Args: cobra.MaximumNArgs(1),
		Annotations: map[string]string{
			mcpannotations.Safe: "false",
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			var err error
			if len(args) == 1 {
				opts.milestoneID, err = strconv.Atoi(args[0])
				if err != nil {
					return err
				}
			}
			return opts.run()
		},
	}

	cmd.Flags().StringVar(&opts.projectID, "project", "", "The ID or URL-encoded path of the project.")
	cmd.Flags().StringVar(&opts.groupID, "group", "", "The ID or URL-encoded path of the group.")

	cmd.Flags().StringVar(&opts.title, "title", "", "Title of the milestone.")
	cmd.Flags().StringVar(&opts.description, "description", "", "Description of the milestone.")
	cmd.Flags().StringVar(&opts.dueDate, "due-date", "", "Due date for the milestone. Expected in ISO 8601 format (2025-04-15T08:00:00Z).")
	cmd.Flags().StringVar(&opts.startDate, "start-date", "", "Start date for the milestone. Expected in ISO 8601 format (2025-04-15T08:00:00Z).")
	cmd.Flags().StringVar(&opts.state, "state", "", "State of the milestone. Can be 'active' or 'close'.")

	return cmd
}

func (o *options) run() error {
	c, err := o.apiClient("")
	if err != nil {
		return err
	}
	client := c.Lab()

	var parsedDueDate, parsedStartDate gitlab.ISOTime

	if o.startDate != "" {
		if parsedStartDate, err = gitlab.ParseISOTime(o.startDate); err != nil {
			return err
		}
	}

	if o.dueDate != "" {
		if parsedDueDate, err = gitlab.ParseISOTime(o.dueDate); err != nil {
			return err
		}
	}

	if o.projectID != "" {
		updateMilestoneOptions := &gitlab.UpdateMilestoneOptions{
			Title:       &o.title,
			Description: &o.description,
		}

		if o.startDate != "" {
			updateMilestoneOptions.StartDate = &parsedStartDate
		}

		if o.dueDate != "" {
			updateMilestoneOptions.DueDate = &parsedDueDate
		}

		if o.state != "" {
			updateMilestoneOptions.StateEvent = &o.state
		}

		milestone, _, err := client.Milestones.UpdateMilestone(o.projectID, o.milestoneID, updateMilestoneOptions)
		if err != nil {
			return err
		}

		o.io.LogInfof("Updated project milestone %s (ID: %d)", milestone.Title, milestone.ID)
		return nil
	} else if o.groupID != "" { // get group milestone
		updateGroupMilestoneOptions := &gitlab.UpdateGroupMilestoneOptions{
			Title:       &o.title,
			Description: &o.description,
		}

		if o.startDate != "" {
			updateGroupMilestoneOptions.StartDate = &parsedStartDate
		}

		if o.dueDate != "" {
			updateGroupMilestoneOptions.DueDate = &parsedDueDate
		}

		if o.state != "" {
			updateGroupMilestoneOptions.StateEvent = &o.state
		}

		milestone, _, err := client.GroupMilestones.UpdateGroupMilestone(o.groupID, o.milestoneID, updateGroupMilestoneOptions)
		if err != nil {
			return err
		}

		o.io.LogInfof("Updated group milestone %s (ID: %d)", milestone.Title, milestone.ID)
		return nil
	}

	// run for the current project
	repo, _ := o.baseRepo()
	updateMilestoneOptions := &gitlab.UpdateMilestoneOptions{
		Title:       &o.title,
		Description: &o.description,
	}

	if o.startDate != "" {
		updateMilestoneOptions.StartDate = &parsedStartDate
	}
	if o.dueDate != "" {
		updateMilestoneOptions.DueDate = &parsedDueDate
	}

	if o.state != "" {
		updateMilestoneOptions.StateEvent = &o.state
	}

	milestone, _, err := client.Milestones.UpdateMilestone(repo.FullName(), o.milestoneID, updateMilestoneOptions)
	if err != nil {
		return err
	}

	o.io.LogInfof("Updated project milestone %s (ID: %d)", milestone.Title, milestone.ID)
	return nil
}
