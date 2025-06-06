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

type IterationListOptions struct {
	Group        string
	Page         int
	PerPage      int
	OutputFormat string
}

func NewCmdList(f cmdutils.Factory) *cobra.Command {
	opts := &IterationListOptions{}

	iterationListCmd := &cobra.Command{
		Use:     "list [flags]",
		Short:   `List project iterations`,
		Long:    ``,
		Aliases: []string{"ls"},
		Example: heredoc.Doc(`
			- glab iteration list
			- glab iteration ls
			- glab iteration list -R owner/repository
			- glab iteration list -g mygroup
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
			iterationApiOpts := &api.ListProjectIterationsOptions{}
			iterationApiOpts.IncludeAncestors = gitlab.Ptr(true)

			if p := opts.Page; p != 0 {
				iterationApiOpts.Page = p
			}
			if pp := opts.PerPage; pp != 0 {
				iterationApiOpts.PerPage = pp
			}

			var iterationBuilder strings.Builder

			if opts.Group != "" {
				iterations, err := api.ListGroupIterations(apiClient, opts.Group, iterationApiOpts)
				if err != nil {
					return err
				}
				if opts.OutputFormat == "json" {
					iterationListJSON, _ := json.Marshal(iterations)
					fmt.Fprintln(f.IO.StdOut, string(iterationListJSON))
				} else {
					fmt.Fprintf(f.IO.StdOut, "Showing iteration %d of %d for group %s.\n\n", len(iterations), len(iterations), opts.Group)
					for _, iteration := range iterations {
						iterationBuilder.WriteString(formatIterationInfo(iteration.Description, iteration.Title, iteration.WebURL))
					}
				}
			} else {
				iterations, err := api.ListProjectIterations(apiClient, repo.FullName(), iterationApiOpts)
				if err != nil {
					return err
				}
				if opts.OutputFormat == "json" {
					iterationListJSON, _ := json.Marshal(iterations)
					fmt.Fprintln(f.IO.StdOut, string(iterationListJSON))
				} else {
					fmt.Fprintf(f.IO.StdOut, "Showing iteration %d of %d on %s.\n\n", len(iterations), len(iterations), repo.FullName())
					for _, iteration := range iterations {
						iterationBuilder.WriteString(formatIterationInfo(iteration.Description, iteration.Title, iteration.WebURL))
					}
				}
			}
			fmt.Fprintln(f.IO.StdOut, utils.Indent(iterationBuilder.String(), " "))
			return nil
		},
	}

	iterationListCmd.Flags().IntVarP(&opts.Page, "page", "p", 1, "Page number.")
	iterationListCmd.Flags().IntVarP(&opts.PerPage, "per-page", "P", 30, "Number of items to list per page.")
	iterationListCmd.Flags().StringVarP(&opts.OutputFormat, "output", "F", "text", "Format output as: text, json.")
	iterationListCmd.Flags().StringVarP(&opts.Group, "group", "g", "", "List iterations for a group.")
	return iterationListCmd
}

func formatIterationInfo(description string, title string, webURL string) string {
	if description != "" {
		description = fmt.Sprintf(" -> %s", description)
	}
	return fmt.Sprintf("%s%s (%s)\n", title, description, webURL)
}
