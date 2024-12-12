package list

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/MakeNowJust/heredoc/v2"
	"gitlab.com/gitlab-org/cli/api"
	"gitlab.com/gitlab-org/cli/commands/cmdutils"
	"gitlab.com/gitlab-org/cli/pkg/utils"

	"github.com/spf13/cobra"
	gitlab "gitlab.com/gitlab-org/api/client-go"
)

type LabelListOptions struct {
	Group        string
	Page         int
	PerPage      int
	OutputFormat string
}

func NewCmdList(f *cmdutils.Factory) *cobra.Command {
	opts := &LabelListOptions{}

	labelListCmd := &cobra.Command{
		Use:     "list [flags]",
		Short:   `List labels in the repository.`,
		Long:    ``,
		Aliases: []string{"ls"},
		Example: heredoc.Doc(`
			glab label list
			glab label ls
			glab label list -R owner/repository
			glab label list -g mygroup
		`),
		Args: cobra.ExactArgs(0),
		RunE: func(cmd *cobra.Command, args []string) error {
			var err error

			apiClient, err := f.HttpClient()
			if err != nil {
				return err
			}

			repo, err := f.BaseRepo()
			if err != nil {
				return err
			}

			labelApiOpts := &api.ListLabelsOptions{}
			labelApiOpts.WithCounts = gitlab.Ptr(true)

			if p := opts.Page; p != 0 {
				labelApiOpts.Page = p
			}
			if pp := opts.PerPage; pp != 0 {
				labelApiOpts.PerPage = pp
			}

			var labelBuilder strings.Builder

			if opts.Group != "" {
				labels, err := api.ListGroupLabels(apiClient, opts.Group, labelApiOpts)
				if err != nil {
					return err
				}
				if opts.OutputFormat == "json" {
					labelListJSON, _ := json.Marshal(labels)
					fmt.Fprintln(f.IO.StdOut, string(labelListJSON))
				} else {
					fmt.Fprintf(f.IO.StdOut, "Showing label %d of %d for group %s.\n\n", len(labels), len(labels), opts.Group)
					for _, label := range labels {
						labelBuilder.WriteString(formatLabelInfo(label.Description, label.Name, label.Color))
					}
				}
			} else {
				labels, err := api.ListLabels(apiClient, repo.FullName(), labelApiOpts)
				if err != nil {
					return err
				}
				if opts.OutputFormat == "json" {
					labelListJSON, _ := json.Marshal(labels)
					fmt.Fprintln(f.IO.StdOut, string(labelListJSON))
				} else {
					fmt.Fprintf(f.IO.StdOut, "Showing label %d of %d on %s.\n\n", len(labels), len(labels), repo.FullName())
					for _, label := range labels {
						labelBuilder.WriteString(formatLabelInfo(label.Description, label.Name, label.Color))
					}
				}

			}
			fmt.Fprintln(f.IO.StdOut, utils.Indent(labelBuilder.String(), " "))
			return nil
		},
	}

	labelListCmd.Flags().IntVarP(&opts.Page, "page", "p", 1, "Page number.")
	labelListCmd.Flags().IntVarP(&opts.PerPage, "per-page", "P", 30, "Number of items to list per page.")
	labelListCmd.Flags().StringVarP(&opts.OutputFormat, "output", "F", "text", "Format output as: text, json.")
	labelListCmd.Flags().StringVarP(&opts.Group, "group", "g", "", "List labels for a group.")

	return labelListCmd
}

func formatLabelInfo(description string, name string, color string) string {
	if description != "" {
		description = fmt.Sprintf(" -> %s", description)
	}
	return fmt.Sprintf("%s%s (%s)\n", name, description, color)
}
