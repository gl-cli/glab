package events

import (
	"encoding/json"
	"fmt"
	"io"

	"github.com/spf13/cobra"
	"github.com/xanzy/go-gitlab"

	"gitlab.com/gitlab-org/cli/api"
	"gitlab.com/gitlab-org/cli/commands/cmdutils"
	"gitlab.com/gitlab-org/cli/pkg/utils"
)

func NewCmdEvents(f *cmdutils.Factory) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "events",
		Short: "View user events.",
		Args:  cobra.ExactArgs(0),
		RunE: func(cmd *cobra.Command, args []string) error {
			apiClient, err := f.HttpClient()
			if err != nil {
				return err
			}

			repo, err := f.BaseRepo()
			if err != nil {
				return err
			}

			l := &gitlab.ListContributionEventsOptions{}

			if p, _ := cmd.Flags().GetInt("page"); p != 0 {
				l.Page = p
			}
			if p, _ := cmd.Flags().GetInt("per-page"); p != 0 {
				l.PerPage = p
			}

			events, err := api.CurrentUserEvents(apiClient, l)
			if err != nil {
				return err
			}

			if err = f.IO.StartPager(); err != nil {
				return err
			}
			defer f.IO.StopPager()

			outputFormat, err := cmd.Flags().GetString("output")
			if err != nil {
				return nil
			}

			if outputFormat != "json" && outputFormat != "text" {
				return fmt.Errorf("--output must be either 'json' or 'text'. Received: %s", outputFormat)
			}

			if outputFormat == "json" {
				return writeJSON(f.IO.StdOut, events)
			}

			if lb, _ := cmd.Flags().GetBool("all"); lb {
				projects := make(map[int]*gitlab.Project)
				for _, e := range events {
					project, err := api.GetProject(apiClient, e.ProjectID)
					if err != nil {
						return err
					}
					projects[e.ProjectID] = project
				}

				title := utils.NewListTitle("User events")
				title.CurrentPageTotal = len(events)

				DisplayAllEvents(f.IO.StdOut, events, projects)
				return nil
			}

			project, err := api.GetProject(apiClient, repo.FullName())
			if err != nil {
				return err
			}

			DisplayProjectEvents(f.IO.StdOut, events, project)
			return nil
		},
	}

	cmd.Flags().BoolP("all", "a", false, "Get events from all projects.")
	cmd.Flags().IntP("page", "p", 1, "Page number.")
	cmd.Flags().IntP("per-page", "P", 30, "Number of items to list per page.")
	cmd.Flags().StringP("output", "F", "text", "Format output as: 'text', 'json'.")
	return cmd
}

func writeJSON(w io.Writer, events []*gitlab.ContributionEvent) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(&events)
}

func DisplayProjectEvents(w io.Writer, events []*gitlab.ContributionEvent, project *gitlab.Project) {
	for _, e := range events {
		if e.ProjectID != project.ID {
			continue
		}
		printEvent(w, e, project)
	}
}

func DisplayAllEvents(w io.Writer, events []*gitlab.ContributionEvent, projects map[int]*gitlab.Project) {
	for _, e := range events {
		printEvent(w, e, projects[e.ProjectID])
	}
}

func printEvent(w io.Writer, e *gitlab.ContributionEvent, project *gitlab.Project) {
	switch e.ActionName {
	case "pushed to":
		fmt.Fprintf(w, "Pushed to %s %s at %s\n%q.\n", e.PushData.RefType, e.PushData.Ref, project.NameWithNamespace, e.PushData.CommitTitle)
	case "deleted":
		fmt.Fprintf(w, "Deleted %s %s at %s.\n", e.PushData.RefType, e.PushData.Ref, project.NameWithNamespace)
	case "pushed new":
		fmt.Fprintf(w, "Pushed new %s %s at %s.\n", e.PushData.RefType, e.PushData.Ref, project.NameWithNamespace)
	case "commented on":
		fmt.Fprintf(w, "Commented on %s #%s at %s.\n%q\n", e.Note.NoteableType, e.Note.Title, project.NameWithNamespace, e.Note.Body)
	case "accepted":
		fmt.Fprintf(w, "Accepted %s %s at %s.\n", e.TargetType, e.TargetTitle, project.NameWithNamespace)
	case "opened":
		fmt.Fprintf(w, "Opened %s %s at %s.\n", e.TargetType, e.TargetTitle, project.NameWithNamespace)
	case "closed":
		fmt.Fprintf(w, "Closed %s %s at %s.\n", e.TargetType, e.TargetTitle, project.NameWithNamespace)
	case "joined":
		fmt.Fprintf(w, "Joined %s.\n", project.NameWithNamespace)
	case "left":
		fmt.Fprintf(w, "Left %s.\n", project.NameWithNamespace)
	case "created":
		targetType := e.TargetType
		if e.TargetType == "WikiPage::Meta" {
			targetType = "Wiki page"
		}
		fmt.Fprintf(w, "Created %s %s at %s.\n", targetType, e.TargetTitle, project.NameWithNamespace)
	default:
		fmt.Fprintf(w, "%s %q", e.TargetType, e.Title)
	}
	fmt.Fprintln(w) // to leave a blank line
}
