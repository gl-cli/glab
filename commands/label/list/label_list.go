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

	labelListCmd := &cobra.Command{
		Use:     "list [flags]",
		Short:   `List labels in the repository.`,
		Long:    ``,
		Aliases: []string{"ls"},
		Example: heredoc.Doc(`
			$ glab label list
			$ glab label ls
			$ glab label list -R owner/repository
			$ glab label list -g mygroup
		`),
		Args: cobra.ExactArgs(0),
		RunE: func(cmd *cobra.Command, args []string) error {
			return opts.run()
		},
	}

	labelListCmd.Flags().IntVarP(&opts.page, "page", "p", 1, "Page number.")
	labelListCmd.Flags().IntVarP(&opts.perPage, "per-page", "P", 30, "Number of items to list per page.")
	labelListCmd.Flags().StringVarP(&opts.outputFormat, "output", "F", "text", "Format output as: text, json.")
	labelListCmd.Flags().StringVarP(&opts.group, "group", "g", "", "List labels for a group.")

	return labelListCmd
}

func (o *options) run() error {
	var err error

	apiClient, err := o.httpClient()
	if err != nil {
		return err
	}

	repo, err := o.baseRepo()
	if err != nil {
		return err
	}

	labelApiOpts := &api.ListLabelsOptions{}
	labelApiOpts.WithCounts = gitlab.Ptr(true)

	if p := o.page; p != 0 {
		labelApiOpts.Page = p
	}
	if pp := o.perPage; pp != 0 {
		labelApiOpts.PerPage = pp
	}

	var labelBuilder strings.Builder

	if o.group != "" {
		labels, err := api.ListGroupLabels(apiClient, o.group, labelApiOpts)
		if err != nil {
			return err
		}
		if o.outputFormat == "json" {
			labelListJSON, _ := json.Marshal(labels)
			fmt.Fprintln(o.io.StdOut, string(labelListJSON))
		} else {
			fmt.Fprintf(o.io.StdOut, "Showing label %d of %d for group %s.\n\n", len(labels), len(labels), o.group)
			for _, label := range labels {
				labelBuilder.WriteString(formatLabelInfo(label.Description, label.Name, label.Color))
			}
		}
	} else {
		labels, err := api.ListLabels(apiClient, repo.FullName(), labelApiOpts)
		if err != nil {
			return err
		}
		if o.outputFormat == "json" {
			labelListJSON, _ := json.Marshal(labels)
			fmt.Fprintln(o.io.StdOut, string(labelListJSON))
		} else {
			fmt.Fprintf(o.io.StdOut, "Showing label %d of %d on %s.\n\n", len(labels), len(labels), repo.FullName())
			for _, label := range labels {
				labelBuilder.WriteString(formatLabelInfo(label.Description, label.Name, label.Color))
			}
		}

	}
	fmt.Fprintln(o.io.StdOut, utils.Indent(labelBuilder.String(), " "))
	return nil
}

func formatLabelInfo(description string, name string, color string) string {
	if description != "" {
		description = fmt.Sprintf(" -> %s", description)
	}
	return fmt.Sprintf("%s%s (%s)\n", name, description, color)
}
