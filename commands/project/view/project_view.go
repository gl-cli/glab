package view

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/url"
	"strings"

	"github.com/MakeNowJust/heredoc/v2"
	"github.com/spf13/cobra"
	"github.com/xanzy/go-gitlab"
	"gitlab.com/gitlab-org/cli/api"
	"gitlab.com/gitlab-org/cli/commands/cmdutils"
	"gitlab.com/gitlab-org/cli/internal/glrepo"
	"gitlab.com/gitlab-org/cli/pkg/iostreams"
	"gitlab.com/gitlab-org/cli/pkg/utils"
)

type ViewOptions struct {
	ProjectID    string
	APIClient    *gitlab.Client
	Web          bool
	OutputFormat string
	Branch       string
	Browser      string
	GlamourStyle string

	IO   *iostreams.IOStreams
	Repo glrepo.Interface
}

func NewCmdView(f *cmdutils.Factory) *cobra.Command {
	opts := ViewOptions{
		IO: f.IO,
	}

	projectViewCmd := &cobra.Command{
		Use:   "view [repository] [flags]",
		Short: "View a project or repository.",
		Long: heredoc.Doc(`Display the description and README of a project, or open it in the browser.
		`),
		Args: cobra.MaximumNArgs(1),
		Example: heredoc.Doc(`
			# View project information for the current directory.
			# Must be a Git repository.
			$ glab repo view

			# View project information of specified name.
			$ glab repo view my-project
			$ glab repo view user/repo
			$ glab repo view group/namespace/repo

			# Specify repository by full [Git] URL.
			$ glab repo view git@gitlab.com:user/repo.git
			$ glab repo view https://gitlab.company.org/user/repo
			$ glab repo view https://gitlab.company.org/user/repo.git
		`),
		RunE: func(cmd *cobra.Command, args []string) error {
			var err error

			cfg, err := f.Config()
			if err != nil {
				return err
			}

			opts.Branch = strings.TrimSpace(opts.Branch)

			if len(args) == 1 {
				opts.ProjectID = args[0]
			}

			// No project argument - use current repository
			if opts.ProjectID == "" {
				opts.Repo, err = f.BaseRepo()
				if err != nil {
					return cmdutils.WrapError(err, "`repository` is required when not running in a Git repository.")
				}

				// Configure client to have host of current repository
				client, err := api.NewClientWithCfg(opts.Repo.RepoHost(), cfg, false)
				if err != nil {
					return err
				}
				opts.APIClient = client.Lab()

				if opts.Branch == "" {
					opts.Branch, _ = f.Branch()
				}
			} else {
				// If the ProjectID is a single token, use current user's namespace
				if !strings.Contains(opts.ProjectID, "/") {
					apiClient, err := f.HttpClient()
					if err != nil {
						return err
					}
					currentUser, err := api.CurrentUser(apiClient)
					if err != nil {
						return cmdutils.WrapError(err, "Failed to retrieve your current user.")
					}

					opts.ProjectID = currentUser.Username + "/" + opts.ProjectID
				}

				// Get the repo full name from the ProjectID which can be a full URL or a group/repo format
				opts.Repo, err = glrepo.FromFullName(opts.ProjectID)
				if err != nil {
					return err
				}
				client, err := api.NewClientWithCfg(opts.Repo.RepoHost(), cfg, false)
				if err != nil {
					return err
				}
				opts.APIClient = client.Lab()
			}

			browser, _ := cfg.Get(opts.Repo.RepoHost(), "browser")
			opts.Browser = browser

			opts.GlamourStyle, _ = cfg.Get(opts.Repo.RepoHost(), "glamour_style")
			return runViewProject(&opts)
		},
	}

	projectViewCmd.Flags().BoolVarP(&opts.Web, "web", "w", false, "Open a project in the browser.")
	projectViewCmd.Flags().StringVarP(&opts.OutputFormat, "output", "F", "text", "Format output as: text, json.")
	projectViewCmd.Flags().StringVarP(&opts.Branch, "branch", "b", "", "View a specific branch of the repository.")

	return projectViewCmd
}

func runViewProject(opts *ViewOptions) error {
	project, err := opts.Repo.Project(opts.APIClient)
	if err != nil {
		return cmdutils.WrapError(err, "Failed to retrieve project information.")
	}

	if opts.Web {
		projectURL := project.WebURL

		if opts.IO.IsaTTY {
			fmt.Fprintf(
				opts.IO.StdOut,
				"Opening %s in your browser.\n",
				generateProjectOpenURL(utils.DisplayURL(projectURL), project.DefaultBranch, opts.Branch),
			)
		}

		return utils.OpenInBrowser(
			generateProjectOpenURL(projectURL, project.DefaultBranch, opts.Branch),
			opts.Browser,
		)
	} else if opts.OutputFormat == "json" {
		printProjectContentJSON(opts, project)
	} else {
		readmeFile, err := getReadmeFile(opts, project)
		if err != nil {
			return err
		}

		if opts.IO.IsaTTY {
			if err := opts.IO.StartPager(); err != nil {
				return err
			}
			defer opts.IO.StopPager()

			printProjectContentTTY(opts, project, readmeFile)
		} else {
			printProjectContentRaw(opts, project, readmeFile)
		}
	}

	return nil
}

func getReadmeFile(opts *ViewOptions, project *gitlab.Project) (*gitlab.File, error) {
	if project.ReadmeURL == "" {
		return nil, nil
	}

	readmePath := strings.Replace(project.ReadmeURL, project.WebURL+"/-/blob/", "", 1)
	readmePathComponents := strings.Split(readmePath, "/")
	readmeRef := readmePathComponents[0]
	readmeFileName := readmePathComponents[1]

	if opts.Branch == "" {
		opts.Branch = readmeRef
	}

	readmeFile, err := api.GetFile(opts.APIClient, project.PathWithNamespace, readmeFileName, opts.Branch)
	if err != nil {
		return nil, cmdutils.WrapError(err, fmt.Sprintf("Failed to retrieve README file on the %s branch.", opts.Branch))
	}

	decoded, err := base64.StdEncoding.DecodeString(readmeFile.Content)
	if err != nil {
		return nil, cmdutils.WrapError(err, "Failed to decode README file.")
	}

	readmeFile.Content = string(decoded)

	return readmeFile, nil
}

func generateProjectOpenURL(projectWebURL string, defaultBranch string, branch string) string {
	if branch != "" && defaultBranch != branch {
		return projectWebURL + "/-/tree/" + url.PathEscape(branch)
	}

	return projectWebURL
}

func printProjectContentTTY(opts *ViewOptions, project *gitlab.Project, readme *gitlab.File) {
	var description string
	var readmeContent string
	var err error

	fullName := project.NameWithNamespace
	if project.Description != "" {
		description, err = utils.RenderMarkdownWithoutIndentations(project.Description, opts.GlamourStyle)
		if err != nil {
			description = project.Description
		}
	} else {
		description = "\n(No description provided)\n\n"
	}

	if readme != nil {
		readmeContent, err = utils.RenderMarkdown(readme.Content, opts.GlamourStyle)
		if err != nil {
			readmeContent = readme.Content
		}
	}

	c := opts.IO.Color()
	// Header
	fmt.Fprint(opts.IO.StdOut, c.Bold(fullName))
	fmt.Fprint(opts.IO.StdOut, c.Gray(description))

	// Readme
	if readme != nil {
		fmt.Fprint(opts.IO.StdOut, readmeContent)
	} else {
		fmt.Fprintln(opts.IO.StdOut, c.Gray("(This repository does not have a README file.)"))
	}

	fmt.Fprintln(opts.IO.StdOut)
	fmt.Fprintf(opts.IO.StdOut, c.Gray("View this project on GitLab: %s\n"), project.WebURL)
}

func printProjectContentRaw(opts *ViewOptions, project *gitlab.Project, readme *gitlab.File) {
	fullName := project.NameWithNamespace
	description := project.Description

	fmt.Fprintf(opts.IO.StdOut, "name:\t%s\n", fullName)
	fmt.Fprintf(opts.IO.StdOut, "description:\t%s\n", description)

	if readme != nil {
		fmt.Fprintln(opts.IO.StdOut, "---")
		fmt.Fprint(opts.IO.StdOut, readme.Content)
		fmt.Fprintln(opts.IO.StdOut)
	}
}

func printProjectContentJSON(opts *ViewOptions, project *gitlab.Project) {
	projectJSON, _ := json.Marshal(project)
	fmt.Fprintln(opts.IO.StdOut, string(projectJSON))
}
