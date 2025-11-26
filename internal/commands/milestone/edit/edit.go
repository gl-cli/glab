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
	milestoneID int64

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
		Args: cobra.ExactArgs(1),
		Annotations: map[string]string{
			mcpannotations.Safe: "false",
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			var err error
			milestoneIDInt, err := strconv.Atoi(args[0])
			if err != nil {
				return err
			}
			opts.milestoneID = int64(milestoneIDInt)
			return opts.run()
		},
	}

	cmd.Flags().StringVar(&opts.projectID, "project", "", "The ID or URL-encoded path of the project.")
	cmd.Flags().StringVar(&opts.groupID, "group", "", "The ID or URL-encoded path of the group.")

	cmd.Flags().StringVar(&opts.title, "title", "", "Title of the milestone.")
	cmd.Flags().StringVar(&opts.description, "description", "", "Description of the milestone.")
	cmd.Flags().StringVar(&opts.dueDate, "due-date", "", "Due date for the milestone. Expected in ISO 8601 format (2025-04-15T08:00:00Z).")
	cmd.Flags().StringVar(&opts.startDate, "start-date", "", "Start date for the milestone. Expected in ISO 8601 format (2025-04-15T08:00:00Z).")
	cmd.Flags().StringVar(&opts.state, "state", "", "State of the milestone. Can be 'activate' or 'close'.")

	return cmd
}

func (o *options) run() error {
	c, err := o.apiClient("")
	if err != nil {
		return err
	}
	client := c.Lab()

	updateMilestoneOptions, updateGroupMilestoneOptions, err := createOptions(o)
	if err != nil {
		return err
	}

	switch {
	case o.projectID != "":
		milestone, _, err := client.Milestones.UpdateMilestone(o.projectID, o.milestoneID, updateMilestoneOptions)
		if err != nil {
			return err
		}
		o.io.LogInfof("Updated project milestone %s (ID: %d)", milestone.Title, milestone.ID)
	case o.groupID != "":
		milestone, _, err := client.GroupMilestones.UpdateGroupMilestone(o.groupID, o.milestoneID, updateGroupMilestoneOptions)
		if err != nil {
			return err
		}
		o.io.LogInfof("Updated group milestone %s (ID: %d)", milestone.Title, milestone.ID)
	default:
		repo, _ := o.baseRepo()
		milestone, _, err := client.Milestones.UpdateMilestone(repo.FullName(), o.milestoneID, updateMilestoneOptions)
		if err != nil {
			return err
		}
		o.io.LogInfof("Updated project milestone %s (ID: %d)", milestone.Title, milestone.ID)

	}
	return nil
}

// createOptions - helper function used to create the UpdateMilestoneOptions and UpdateGroupMilestoneOptions
func createOptions(o *options) (*gitlab.UpdateMilestoneOptions, *gitlab.UpdateGroupMilestoneOptions, error) {
	var parsedDueDate, parsedStartDate gitlab.ISOTime
	var err error

	if o.startDate != "" {
		if parsedStartDate, err = gitlab.ParseISOTime(o.startDate); err != nil {
			return nil, nil, err
		}
	}

	if o.dueDate != "" {
		if parsedDueDate, err = gitlab.ParseISOTime(o.dueDate); err != nil {
			return nil, nil, err
		}
	}

	updateMilestoneOptions := &gitlab.UpdateMilestoneOptions{}
	updateGroupMilestoneOptions := &gitlab.UpdateGroupMilestoneOptions{}

	if o.title != "" {
		updateMilestoneOptions.Title = &o.title
		updateGroupMilestoneOptions.Title = &o.title
	}
	if o.description != "" {
		updateMilestoneOptions.Description = &o.description
		updateGroupMilestoneOptions.Description = &o.description
	}

	if o.startDate != "" {
		updateMilestoneOptions.StartDate = &parsedStartDate
		updateGroupMilestoneOptions.StartDate = &parsedStartDate
	}

	if o.dueDate != "" {
		updateMilestoneOptions.DueDate = &parsedDueDate
		updateGroupMilestoneOptions.DueDate = &parsedDueDate
	}

	if o.state != "" {
		updateMilestoneOptions.StateEvent = &o.state
		updateGroupMilestoneOptions.StateEvent = &o.state
	}

	return updateMilestoneOptions, updateGroupMilestoneOptions, nil
}
