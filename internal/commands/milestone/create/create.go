package create

import (
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

	projectID string
	groupID   string

	title       string
	description string
	dueDate     string
	startDate   string
}

func NewCmdCreate(f cmdutils.Factory) *cobra.Command {
	opts := &options{
		io:        f.IO(),
		apiClient: f.ApiClient,
		baseRepo:  f.BaseRepo,
	}

	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create a group or project milestone.",
		Long:  "",
		Example: heredoc.Doc(`
		  # Create milestone for the current project
			$ glab milestone create --title='Example title' --due-date='2025-12-16'

			# Create milestone for the specified project
			$ glab milestone create --title='Example group milestone' --due-date='2025-12-16' --project 123

			# Create milestone for the specified group
			$ glab milestone create --title='Example group milestone' --due-date='2025-12-16' --group 456
		`),
		Args: cobra.MaximumNArgs(0),
		Annotations: map[string]string{
			mcpannotations.Safe: "false",
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			return opts.run()
		},
	}

	cmd.Flags().StringVar(&opts.projectID, "project", "", "The ID or URL-encoded path of the project.")
	cmd.Flags().StringVar(&opts.groupID, "group", "", "The ID or URL-encoded path of the group.")

	cmd.Flags().StringVar(&opts.title, "title", "", "Title of the milestone.")
	cmd.Flags().StringVar(&opts.description, "description", "", "Description of the milestone.")
	cmd.Flags().StringVar(&opts.dueDate, "due-date", "", "Due date for the milestone. Expected in ISO 8601 format (2025-04-15T08:00:00Z).")
	cmd.Flags().StringVar(&opts.startDate, "start-date", "", "Start date for the milestone. Expected in ISO 8601 format (2025-04-15T08:00:00Z).")

	cobra.CheckErr(cmd.MarkFlagRequired("title"))

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
		createMilestoneOptions := &gitlab.CreateMilestoneOptions{
			Title:       &o.title,
			Description: &o.description,
		}

		if o.startDate != "" {
			createMilestoneOptions.StartDate = &parsedStartDate
		}

		if o.dueDate != "" {
			createMilestoneOptions.DueDate = &parsedDueDate
		}

		milestone, _, err := client.Milestones.CreateMilestone(o.projectID, createMilestoneOptions)
		if err != nil {
			return err
		}

		o.io.LogInfof("Created project milestone %s (ID: %d)", milestone.Title, milestone.ID)
		return nil
	} else if o.groupID != "" { // get group milestone
		createGroupMilestoneOptions := &gitlab.CreateGroupMilestoneOptions{
			Title:       &o.title,
			Description: &o.description,
		}

		if o.startDate != "" {
			createGroupMilestoneOptions.StartDate = &parsedStartDate
		}

		if o.dueDate != "" {
			createGroupMilestoneOptions.DueDate = &parsedDueDate
		}

		milestone, _, err := client.GroupMilestones.CreateGroupMilestone(o.groupID, createGroupMilestoneOptions)
		if err != nil {
			return err
		}

		o.io.LogInfof("Created group milestone %s (ID: %d)", milestone.Title, milestone.ID)
		return nil
	}

	// run for the current project
	repo, _ := o.baseRepo()
	createMilestoneOptions := &gitlab.CreateMilestoneOptions{
		Title:       &o.title,
		Description: &o.description,
	}

	if o.startDate != "" {
		createMilestoneOptions.StartDate = &parsedStartDate
	}
	if o.dueDate != "" {
		createMilestoneOptions.DueDate = &parsedDueDate
	}

	milestone, _, err := client.Milestones.CreateMilestone(repo.FullName(), createMilestoneOptions)
	if err != nil {
		return err
	}

	o.io.LogInfof("Created project milestone %s (ID: %d)", milestone.Title, milestone.ID)
	return nil
}
