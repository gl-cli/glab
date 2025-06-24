package list

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/MakeNowJust/heredoc/v2"
	"gitlab.com/gitlab-org/cli/api"
	"gitlab.com/gitlab-org/cli/commands/cmdutils"
	"gitlab.com/gitlab-org/cli/internal/glrepo"
	"gitlab.com/gitlab-org/cli/pkg/iostreams"
	"gitlab.com/gitlab-org/cli/pkg/utils"

	"github.com/spf13/cobra"
	gitlab "gitlab.com/gitlab-org/api/client-go"
)

type options struct {
	io           *iostreams.IOStreams
	httpClient   func() (*gitlab.Client, error)
	baseRepo     func() (glrepo.Interface, error)
	group        string
	page         int
	perPage      int
	outputFormat string
}

func NewCmdList(f cmdutils.Factory) *cobra.Command {
	opts := &options{
		io:         f.IO(),
		httpClient: f.HttpClient,
		baseRepo:   f.BaseRepo,
	}

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
			return opts.run()
		},
	}

	iterationListCmd.Flags().IntVarP(&opts.page, "page", "p", 1, "Page number.")
	iterationListCmd.Flags().IntVarP(&opts.perPage, "per-page", "P", 30, "Number of items to list per page.")
	iterationListCmd.Flags().StringVarP(&opts.outputFormat, "output", "F", "text", "Format output as: text, json.")
	iterationListCmd.Flags().StringVarP(&opts.group, "group", "g", "", "List iterations for a group.")
	return iterationListCmd
}

func (o *options) run() error {
	apiClient, err := o.httpClient()
	if err != nil {
		return err
	}
	repo, err := o.baseRepo()
	if err != nil {
		return err
	}
	iterationApiOpts := &api.ListProjectIterationsOptions{}
	iterationApiOpts.IncludeAncestors = gitlab.Ptr(true)

	if p := o.page; p != 0 {
		iterationApiOpts.Page = p
	}
	if pp := o.perPage; pp != 0 {
		iterationApiOpts.PerPage = pp
	}

	var iterationBuilder strings.Builder

	if o.group != "" {
		iterations, err := api.ListGroupIterations(apiClient, o.group, iterationApiOpts)
		if err != nil {
			return err
		}
		if o.outputFormat == "json" {
			iterationListJSON, _ := json.Marshal(iterations)
			fmt.Fprintln(o.io.StdOut, string(iterationListJSON))
		} else {
			fmt.Fprintf(o.io.StdOut, "Showing iteration %d of %d for group %s.\n\n", len(iterations), len(iterations), o.group)
			for _, iteration := range iterations {
				iterationBuilder.WriteString(formatIterationInfo(iteration.Description, iteration.Title, iteration.WebURL))
			}
		}
	} else {
		iterations, err := api.ListProjectIterations(apiClient, repo.FullName(), iterationApiOpts)
		if err != nil {
			return err
		}
		if o.outputFormat == "json" {
			iterationListJSON, _ := json.Marshal(iterations)
			fmt.Fprintln(o.io.StdOut, string(iterationListJSON))
		} else {
			fmt.Fprintf(o.io.StdOut, "Showing iteration %d of %d on %s.\n\n", len(iterations), len(iterations), repo.FullName())
			for _, iteration := range iterations {
				iterationBuilder.WriteString(formatIterationInfo(iteration.Description, iteration.Title, iteration.WebURL))
			}
		}
	}
	fmt.Fprintln(o.io.StdOut, utils.Indent(iterationBuilder.String(), " "))
	return nil
}

func formatIterationInfo(description string, title string, webURL string) string {
	if description != "" {
		description = fmt.Sprintf(" -> %s", description)
	}
	return fmt.Sprintf("%s%s (%s)\n", title, description, webURL)
}
