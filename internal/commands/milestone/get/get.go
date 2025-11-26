package get

import (
	"fmt"
	"strconv"

	"github.com/MakeNowJust/heredoc/v2"
	"github.com/spf13/cobra"

	"gitlab.com/gitlab-org/cli/internal/api"
	"gitlab.com/gitlab-org/cli/internal/cmdutils"
	"gitlab.com/gitlab-org/cli/internal/glrepo"
	"gitlab.com/gitlab-org/cli/internal/iostreams"
	"gitlab.com/gitlab-org/cli/internal/mcpannotations"
	"gitlab.com/gitlab-org/cli/internal/utils"
)

type options struct {
	apiClient func(repoHost string) (*api.Client, error)
	io        *iostreams.IOStreams
	baseRepo  func() (glrepo.Interface, error)

	projectID   string
	groupID     string
	milestoneID int64
}

func NewCmdGet(f cmdutils.Factory) *cobra.Command {
	opts := &options{
		io:        f.IO(),
		apiClient: f.ApiClient,
		baseRepo:  f.BaseRepo,
	}
	cmd := &cobra.Command{
		Use:   "get",
		Short: "Get a milestones via an ID for a project or group.",
		Long:  "",
		Example: heredoc.Doc(`
		  # Get milestone for the current project
			$ glab milestone get 123

			# Get milestone for the specified project
			$ glab milestone get 123 --project project-name

			# Get milestone for the specified group
			$ glab milestone get 123 --group group-name
		`),
		Args: cobra.MaximumNArgs(1),
		Annotations: map[string]string{
			mcpannotations.Safe: "true",
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 1 {
				milestoneIDInt, err := strconv.Atoi(args[0])
				if err != nil {
					return err
				}
				opts.milestoneID = int64(milestoneIDInt)
			}

			return opts.run()
		},
	}

	cmd.Flags().StringVar(&opts.projectID, "project", "", "The ID or URL-encoded path of the project.")
	cmd.Flags().StringVar(&opts.groupID, "group", "", "The ID or URL-encoded path of the group.")

	return cmd
}

func (o *options) run() error {
	c, err := o.apiClient("")
	if err != nil {
		return err
	}
	client := c.Lab()

	if o.projectID != "" { // get project milestone
		milestone, _, err := client.Milestones.GetMilestone(o.projectID, o.milestoneID)
		if err != nil {
			return err
		}

		o.io.LogInfo(fmt.Sprintf("Title: %s\nDescription: %s\nState: %s\nDue Date: %s\n", milestone.Title, milestone.Description, milestone.State, utils.FormatDueDate(milestone.DueDate)))
		return nil
	} else if o.groupID != "" { // get group milestone
		milestone, _, err := client.GroupMilestones.GetGroupMilestone(o.groupID, o.milestoneID)
		if err != nil {
			return err
		}

		o.io.LogInfo(fmt.Sprintf("Title: %s\nDescription: %s\nState: %s\nDue Date: %s\n", milestone.Title, milestone.Description, milestone.State, utils.FormatDueDate(milestone.DueDate)))
		return nil
	}

	// run for the current project
	repo, _ := o.baseRepo()
	milestone, _, err := client.Milestones.GetMilestone(repo.FullName(), o.milestoneID)
	if err != nil {
		return err
	}

	o.io.LogInfo(fmt.Sprintf("Title: %s\nDescription: %s\nState: %s\nDue Date: %s\n", milestone.Title, milestone.Description, milestone.State, utils.FormatDueDate(milestone.DueDate)))
	return nil
}
