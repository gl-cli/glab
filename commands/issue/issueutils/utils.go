package issueutils

import (
	"context"
	"fmt"
	"net/url"
	"regexp"
	"strconv"
	"strings"

	"gitlab.com/gitlab-org/cli/pkg/iostreams"

	"gitlab.com/gitlab-org/cli/api"
	"gitlab.com/gitlab-org/cli/internal/config"
	"gitlab.com/gitlab-org/cli/internal/glrepo"
	"golang.org/x/sync/errgroup"

	"gitlab.com/gitlab-org/cli/pkg/tableprinter"
	"gitlab.com/gitlab-org/cli/pkg/utils"

	"github.com/xanzy/go-gitlab"
)

func DisplayIssueList(streams *iostreams.IOStreams, issues []*gitlab.Issue, projectID string) string {
	c := streams.Color()
	table := tableprinter.NewTablePrinter()
	table.SetIsTTY(streams.IsOutputTTY())
	for _, issue := range issues {
		table.AddCell(streams.Hyperlink(IssueState(c, issue), issue.WebURL))
		// list of MR closing issues doesn't contain references
		// https://gitlab.com/gitlab-org/cli/-/merge_requests/1092
		if issue.References != nil {
			table.AddCell(issue.References.Full)
		}
		table.AddCell(issue.Title)

		if len(issue.Labels) > 0 {
			table.AddCellf("(%s)", c.Cyan(strings.Trim(strings.Join(issue.Labels, ", "), ",")))
		} else {
			table.AddCell("")
		}

		table.AddCell(c.Gray(utils.TimeToPrettyTimeAgo(*issue.CreatedAt)))
		table.EndRow()
	}

	return table.Render()
}

func DisplayIssue(c *iostreams.ColorPalette, i *gitlab.Issue, isTTY bool) string {
	duration := utils.TimeToPrettyTimeAgo(*i.CreatedAt)
	issueID := IssueState(c, i)

	if isTTY {
		return fmt.Sprintf("%s %s (%s)\n %s\n", issueID, i.Title, duration, i.WebURL)
	} else {
		return i.WebURL
	}
}

func IssueState(c *iostreams.ColorPalette, i *gitlab.Issue) (issueID string) {
	if i.State == "opened" {
		issueID = c.Green(fmt.Sprintf("#%d", i.IID))
	} else {
		issueID = c.Red(fmt.Sprintf("#%d", i.IID))
	}
	return
}

func IssuesFromArgs(apiClient *gitlab.Client, baseRepoFn func() (glrepo.Interface, error), args []string) ([]*gitlab.Issue, glrepo.Interface, error) {
	var baseRepo glrepo.Interface

	if len(args) <= 1 {
		if len(args) == 1 {
			args = strings.Split(args[0], ",")
		}
		if len(args) <= 1 {
			issue, repo, err := IssueFromArg(apiClient, baseRepoFn, args[0])
			if err != nil {
				return nil, nil, err
			}
			baseRepo = repo
			return []*gitlab.Issue{issue}, baseRepo, err
		}
	}

	errGroup, _ := errgroup.WithContext(context.Background())
	issues := make([]*gitlab.Issue, len(args))
	for i, arg := range args {
		i, arg := i, arg
		errGroup.Go(func() error {
			issue, repo, err := IssueFromArg(apiClient, baseRepoFn, arg)
			if err != nil {
				return err
			}
			baseRepo = repo
			issues[i] = issue
			return nil
		})
	}
	if err := errGroup.Wait(); err != nil {
		return nil, nil, err
	}
	return issues, baseRepo, nil
}

func IssueFromArg(apiClient *gitlab.Client, baseRepoFn func() (glrepo.Interface, error), arg string) (*gitlab.Issue, glrepo.Interface, error) {
	issueIID, baseRepo := issueMetadataFromURL(arg)
	if issueIID == 0 {
		var err error
		issueIID, err = strconv.Atoi(strings.TrimPrefix(arg, "#"))
		if err != nil {
			return nil, nil, fmt.Errorf("invalid issue format: %q", arg)
		}
	}

	if baseRepo == nil {
		var err error
		baseRepo, err = baseRepoFn()
		if err != nil {
			return nil, nil, fmt.Errorf("could not determine base repository: %w", err)
		}
	} else {
		// initialize a new HTTP Client with the new host
		// TODO: avoid reinitializing the config, get the config as a parameter

		cfg, _ := config.Init()
		a, err := api.NewClientWithCfg(baseRepo.RepoHost(), cfg, false)
		if err != nil {
			return nil, nil, err
		}
		apiClient = a.Lab()
	}

	issue, err := issueFromIID(apiClient, baseRepo, issueIID)
	return issue, baseRepo, err
}

// issueURLPathRE is a regex which matches the following patterns:
//
//		OWNER/REPO/issues/id
//		OWNER/REPO/-/issues/id
//		OWNER/REPO/-/issues/incident/id
//		GROUP/NAMESPACE/REPO/issues/id
//		GROUP/NAMESPACE/REPO/-/issues/id
//		GROUP/NAMESPACE/REPO/-/issues/incident/id
//	including nested subgroups:
//		GROUP/SUBGROUP/../../REPO/-/issues/id
//		GROUP/SUBGROUP/../../REPO/-/issues/incident/id
var issueURLPathRE = regexp.MustCompile(`^(/(?:[^-][^/]+/){2,})+(?:-/)?issues/(?:incident/)?(\d+)$`)

func issueMetadataFromURL(s string) (int, glrepo.Interface) {
	u, err := url.Parse(s)
	if err != nil {
		return 0, nil
	}

	if u.Scheme != "https" && u.Scheme != "http" {
		return 0, nil
	}

	m := issueURLPathRE.FindStringSubmatch(u.Path)
	if m == nil {
		return 0, nil
	}

	issueIID, err := strconv.Atoi(m[2])
	if err != nil {
		return 0, nil
	}
	u.Path = m[1]

	repo, err := glrepo.FromURL(u)
	if err != nil {
		return 0, nil
	}
	return issueIID, repo
}

func issueFromIID(apiClient *gitlab.Client, repo glrepo.Interface, issueIID int) (*gitlab.Issue, error) {
	return api.GetIssue(apiClient, repo.FullName(), issueIID)
}
