package list

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/MakeNowJust/heredoc/v2"
	"gitlab.com/gitlab-org/cli/internal/api"
	"gitlab.com/gitlab-org/cli/internal/cmdutils"
	"gitlab.com/gitlab-org/cli/internal/config"
	"gitlab.com/gitlab-org/cli/internal/glrepo"
	"gitlab.com/gitlab-org/cli/internal/iostreams"
	"gitlab.com/gitlab-org/cli/internal/utils"

	"github.com/spf13/cobra"
	gitlab "gitlab.com/gitlab-org/api/client-go"
)

type options struct {
	io           *iostreams.IOStreams
	apiClient    func(repoHost string, cfg config.Config) (*api.Client, error)
	config       config.Config
	baseRepo     func() (glrepo.Interface, error)
	group        string
	page         int
	perPage      int
	outputFormat string
}

func NewCmdList(f cmdutils.Factory) *cobra.Command {
	opts := &options{
		io:        f.IO(),
		apiClient: f.ApiClient,
		config:    f.Config(),
		baseRepo:  f.BaseRepo,
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

type listLabelsOptions struct {
	withCounts *bool
	perPage    int
	page       int
}

func (opts *listLabelsOptions) listLabelsOptions() *gitlab.ListLabelsOptions {
	projectOpts := &gitlab.ListLabelsOptions{}
	projectOpts.WithCounts = opts.withCounts
	projectOpts.PerPage = opts.perPage
	projectOpts.Page = opts.page
	return projectOpts
}

func (opts *listLabelsOptions) listGroupLabelsOptions() *gitlab.ListGroupLabelsOptions {
	groupOpts := &gitlab.ListGroupLabelsOptions{}
	groupOpts.WithCounts = opts.withCounts
	groupOpts.PerPage = opts.perPage
	groupOpts.Page = opts.page
	return groupOpts
}

func (o *options) run() error {
	var err error

	// NOTE: this command can not only be used for projects,
	// so we have to manually check for the base repo, it it doesn't exist,
	// we bootstrap the client with the default hostname.
	var repoHost string
	if baseRepo, err := o.baseRepo(); err == nil {
		repoHost = baseRepo.RepoHost()
	}
	apiClient, err := o.apiClient(repoHost, o.config)
	if err != nil {
		return err
	}
	client := apiClient.Lab()

	labelApiOpts := &listLabelsOptions{}
	labelApiOpts.withCounts = gitlab.Ptr(true)

	if o.page != 0 {
		labelApiOpts.page = o.page
	}
	if o.perPage != 0 {
		labelApiOpts.perPage = o.perPage
	} else {
		labelApiOpts.perPage = api.DefaultListLimit
	}

	var labelBuilder strings.Builder

	if o.group != "" {
		labels, _, err := client.GroupLabels.ListGroupLabels(o.group, labelApiOpts.listGroupLabelsOptions())
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
		repo, err := o.baseRepo()
		if err != nil {
			return err
		}

		labels, _, err := client.Labels.ListLabels(repo.FullName(), labelApiOpts.listLabelsOptions())
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
