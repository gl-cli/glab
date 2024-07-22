package list

import (
	"encoding/json"
	"fmt"

	"github.com/MakeNowJust/heredoc/v2"
	"gitlab.com/gitlab-org/cli/api"

	"gitlab.com/gitlab-org/cli/commands/cmdutils"
	"gitlab.com/gitlab-org/cli/pkg/utils"

	"github.com/spf13/cobra"
	"github.com/xanzy/go-gitlab"
)

var OutputFormat string

func NewCmdList(f *cmdutils.Factory) *cobra.Command {
	labelListCmd := &cobra.Command{
		Use:     "list [flags]",
		Short:   `List labels in the repository.`,
		Long:    ``,
		Aliases: []string{"ls"},
		Example: heredoc.Doc(`
			glab label list
			glab label ls
			glab label list -R owner/repository
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

			l := &gitlab.ListLabelsOptions{}

			l.WithCounts = gitlab.Ptr(true)

			if p, _ := cmd.Flags().GetInt("page"); p != 0 {
				l.Page = p
			}
			if p, _ := cmd.Flags().GetInt("per-page"); p != 0 {
				l.PerPage = p
			}

			// List all labels
			labels, err := api.ListLabels(apiClient, repo.FullName(), l)
			if err != nil {
				return err
			}
			if OutputFormat == "json" {
				labelListJSON, _ := json.Marshal(labels)
				fmt.Fprintln(f.IO.StdOut, string(labelListJSON))

			} else {
				fmt.Fprintf(f.IO.StdOut, "Showing label %d of %d on %s.\n\n", len(labels), len(labels), repo.FullName())
				var labelPrintInfo string
				for _, label := range labels {
					var description string
					if label.Description != "" {
						description = fmt.Sprintf(" -> %s", label.Description)
					}
					labelPrintInfo += fmt.Sprintf("%s%s (%s)\n", label.Name, description, label.Color)
				}
				fmt.Fprintln(f.IO.StdOut, utils.Indent(labelPrintInfo, " "))
			}

			// Cache labels for host
			//labelNames := make([]string, 0, len(labels))
			//for _, label := range labels {
			//	labelNames = append(labelNames, label.Name)
			//}
			//labelsEntry := strings.Join(labelNames, ",")
			//if err := cfg.Set(repo.RepoHost(), "project_labels", labelsEntry); err != nil {
			//	_ = cfg.Write()
			//}

			return nil
		},
	}

	labelListCmd.Flags().IntP("page", "p", 1, "Page number.")
	labelListCmd.Flags().IntP("per-page", "P", 30, "Number of items to list per page.")
	labelListCmd.Flags().StringVarP(&OutputFormat, "output", "F", "text", "Format output as: text, json.")

	return labelListCmd
}
