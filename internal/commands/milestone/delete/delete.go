package delete

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
)

type options struct {
	apiClient func(repoHost string) (*api.Client, error)
	io        *iostreams.IOStreams
	baseRepo  func() (glrepo.Interface, error)

	projectID   string
	groupID     string
	milestoneID int64
}

func NewCmdDelete(f cmdutils.Factory) *cobra.Command {
	opts := &options{
		io:        f.IO(),
		apiClient: f.ApiClient,
		baseRepo:  f.BaseRepo,
	}
	cmd := &cobra.Command{
		Use:   "delete",
		Short: "Delete a group or project milestone.",
		Long:  "",
		Example: heredoc.Doc(`
			# Delete milestone for the current project
			$ glab milestone delete 123

			# Delete milestone for the specified project
			$ glab milestone delete 123 --project project-name

			# Delete milestone for the specified group
			$ glab milestone delete 123 --group group-name
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

	return cmd
}

func (o *options) run() error {
	c, err := o.apiClient("")
	if err != nil {
		return err
	}
	client := c.Lab()

	switch {
	case o.projectID != "":
		_, err := client.Milestones.DeleteMilestone(o.projectID, o.milestoneID)
		if err != nil {
			return err
		}

		o.io.LogInfo(fmt.Sprintf("Deleted project milestone with ID %d.", o.milestoneID))
	case o.groupID != "":
		_, err := client.GroupMilestones.DeleteGroupMilestone(o.groupID, o.milestoneID)
		if err != nil {
			return err
		}

		o.io.LogInfo(fmt.Sprintf("Deleted group milestone with ID %d.", o.milestoneID))
	default:
		repo, _ := o.baseRepo()
		_, err = client.Milestones.DeleteMilestone(repo.FullName(), o.milestoneID)
		if err != nil {
			return err
		}

		o.io.LogInfo(fmt.Sprintf("Deleted project milestone with ID %d.", o.milestoneID))
	}
	return nil
}
