package ciutils

import (
	"context"
	"fmt"
	"io"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/AlecAivazis/survey/v2"
	"github.com/AlecAivazis/survey/v2/terminal"
	"github.com/pkg/errors"

	gitlab "gitlab.com/gitlab-org/api/client-go"

	"gitlab.com/gitlab-org/cli/internal/api"
	"gitlab.com/gitlab-org/cli/internal/glrepo"
	"gitlab.com/gitlab-org/cli/internal/iostreams"
	"gitlab.com/gitlab-org/cli/internal/prompt"
	"gitlab.com/gitlab-org/cli/internal/tableprinter"
	"gitlab.com/gitlab-org/cli/internal/utils"
)

func makeHyperlink(s *iostreams.IOStreams, pipeline *gitlab.PipelineInfo) string {
	return s.Hyperlink(fmt.Sprintf("%d", pipeline.ID), pipeline.WebURL)
}

func DisplaySchedules(i *iostreams.IOStreams, s []*gitlab.PipelineSchedule, projectID string) string {
	if len(s) > 0 {
		table := tableprinter.NewTablePrinter()
		table.AddRow("ID", "Description", "Cron", "Owner", "Active")
		for _, schedule := range s {
			table.AddRow(schedule.ID, schedule.Description, schedule.Cron, schedule.Owner.Username, schedule.Active)
		}

		return table.Render()
	}

	// return empty string, since when there is no schedule, the title will already display it accordingly
	return ""
}

func DisplayMultiplePipelines(s *iostreams.IOStreams, p []*gitlab.PipelineInfo, projectID string) string {
	c := s.Color()

	table := tableprinter.NewTablePrinter()

	if len(p) > 0 {
		table.AddRow("State", "IID", "Ref", "Created")
		for _, pipeline := range p {
			duration := ""

			if pipeline.CreatedAt != nil {
				duration = c.Magenta("(" + utils.TimeToPrettyTimeAgo(*pipeline.CreatedAt) + ")")
			}

			var pipeState string
			switch pipeline.Status {
			case "success":
				pipeState = c.Green(fmt.Sprintf("(%s) • #%s", pipeline.Status, makeHyperlink(s, pipeline)))
			case "failed":
				pipeState = c.Red(fmt.Sprintf("(%s) • #%s", pipeline.Status, makeHyperlink(s, pipeline)))
			default:
				pipeState = c.Gray(fmt.Sprintf("(%s) • #%s", pipeline.Status, makeHyperlink(s, pipeline)))
			}

			table.AddRow(pipeState, fmt.Sprintf("(#%d)", pipeline.IID), pipeline.Ref, duration)
		}

		return table.Render()
	}

	return "No Pipelines available on " + projectID
}

func RunTraceSha(ctx context.Context, apiClient *gitlab.Client, w io.Writer, pid any, sha, name string) error {
	job, err := api.PipelineJobWithSha(apiClient, pid, sha, name)
	if err != nil || job == nil {
		return errors.Wrap(err, "failed to find job")
	}
	return runTrace(ctx, apiClient, w, pid, job.ID)
}

func runTrace(ctx context.Context, apiClient *gitlab.Client, w io.Writer, pid any, jobId int64) error {
	var once sync.Once
	var offset int64

	fmt.Fprintln(w, "Getting job trace...")
	for range time.NewTicker(time.Second * 3).C {
		if ctx.Err() == context.Canceled {
			break
		}
		job, _, err := apiClient.Jobs.GetJob(pid, jobId)
		if err != nil {
			return errors.Wrap(err, "failed to find job")
		}
		switch job.Status {
		case "pending":
			fmt.Fprintf(w, "%s is pending... waiting for job to start.\n", job.Name)
			continue
		case "manual":
			fmt.Fprintf(w, "Manual job %s not started, waiting for job to start.\n", job.Name)
			continue
		case "skipped":
			fmt.Fprintf(w, "%s has been skipped.\n", job.Name)
		}
		once.Do(func() {
			fmt.Fprintf(w, "Showing logs for %s job #%d.\n", job.Name, job.ID)
		})
		trace, _, err := apiClient.Jobs.GetTraceFile(pid, jobId)
		if err != nil || trace == nil {
			return errors.Wrap(err, "failed to find job")
		}
		_, _ = io.CopyN(io.Discard, trace, offset)
		lenT, err := io.Copy(w, trace)
		if err != nil {
			return err
		}
		offset += lenT

		if job.Status == "success" ||
			job.Status == "failed" ||
			job.Status == "cancelled" {
			return nil
		}
	}
	return nil
}

func GetJobId(inputs *JobInputs, opts *JobOptions) (int64, error) {
	// If the user hasn't supplied an argument, we display the jobs list interactively.
	if inputs.JobName == "" {
		return getJobIdInteractive(inputs, opts)
	}

	// If the user supplied a job ID, we can use it directly.
	if jobID, err := strconv.Atoi(inputs.JobName); err == nil {
		return int64(jobID), nil
	}

	// Otherwise, we try to find the latest job ID based on the job name.
	pipelineId, err := getPipelineId(inputs, opts)
	if err != nil {
		return 0, fmt.Errorf("get pipeline: %w", err)
	}

	// This is also the default
	jobs := make([]*gitlab.Job, 0)
	options := &gitlab.ListJobsOptions{
		ListOptions: gitlab.ListOptions{
			PerPage: 20,
			Page:    1,
		},
	}

	for {
		jobsPerPage, response, err := opts.Client.Jobs.ListPipelineJobs(opts.Repo.FullName(), pipelineId, options)
		if err != nil {
			return 0, fmt.Errorf("list pipeline jobs: %w", err)
		}
		jobs = append(jobs, jobsPerPage...)

		// indicate that we have reached the last page
		if response.NextPage == 0 {
			break
		}

		options.Page = response.NextPage
	}

	if len(jobs) == 0 {
		return 0, fmt.Errorf("pipeline %d contains no jobs at all", pipelineId)
	}

	for _, job := range jobs {
		if job.Name == inputs.JobName {
			return job.ID, nil
		}
	}

	return 0, fmt.Errorf("pipeline %d contains no jobs with the name %s", pipelineId, inputs.JobName)
}

func getPipelineId(inputs *JobInputs, opts *JobOptions) (int64, error) {
	if inputs.PipelineId != 0 {
		return int64(inputs.PipelineId), nil
	}

	branch := GetBranch(inputs.Branch, nil, opts.Repo, opts.Client)
	if branch == "" {
		return 0, fmt.Errorf("unable to determine branch")
	}

	pipeline, _, err := opts.Client.Pipelines.GetLatestPipeline(opts.Repo.FullName(), &gitlab.GetLatestPipelineOptions{Ref: gitlab.Ptr(branch)})
	if err != nil {
		return 0, fmt.Errorf("get last pipeline: %w", err)
	}
	return pipeline.ID, err
}

// GetDefaultBranch fetches the repository's default branch from GitLab API.
// Falls back to "main" if the API call fails or returns empty.
func GetDefaultBranch(repo glrepo.Interface, client *gitlab.Client) string {
	if repo == nil || client == nil {
		return "main"
	}
	project, _, err := client.Projects.GetProject(repo.FullName(), nil)
	if err != nil {
		return "main"
	}
	if project.DefaultBranch != "" {
		return project.DefaultBranch
	}
	return "main"
}

// GetBranch returns the specified branch, current git branch, or the default branch from API
func GetBranch(branch string, currentBranch func() (string, error), repo glrepo.Interface, client *gitlab.Client) string {
	if branch != "" {
		return branch
	}
	if currentBranch != nil {
		if gitBranch, _ := currentBranch(); gitBranch != "" {
			return gitBranch
		}
	}
	return GetDefaultBranch(repo, client)
}

func getJobIdInteractive(inputs *JobInputs, opts *JobOptions) (int64, error) {
	pipelineId, err := getPipelineId(inputs, opts)
	if err != nil {
		return 0, err
	}

	fmt.Fprintf(opts.IO.StdOut, "Getting jobs for pipeline %d...\n\n", pipelineId)

	listOptions := &gitlab.ListJobsOptions{
		ListOptions: gitlab.ListOptions{
			PerPage: 100,
		},
	}
	jobs, err := gitlab.ScanAndCollect(func(p gitlab.PaginationOptionFunc) ([]*gitlab.Job, *gitlab.Response, error) {
		return opts.Client.Jobs.ListPipelineJobs(opts.Repo.FullName(), pipelineId, listOptions)
	})
	if err != nil {
		return 0, err
	}

	var jobOptions []string
	var selectedJob string

	for _, job := range jobs {
		if inputs.SelectionPredicate == nil || inputs.SelectionPredicate(job) {
			jobOptions = append(jobOptions, fmt.Sprintf("%s (%d) - %s", job.Name, job.ID, job.Status))
		}
	}

	messagePrompt := inputs.SelectionPrompt
	if messagePrompt == "" {
		messagePrompt = "Select pipeline job to trace:"
	}

	promptOpts := &survey.Select{
		Message: messagePrompt,
		Options: jobOptions,
	}
	if len(jobOptions) > 0 {

		err = prompt.AskOne(promptOpts, &selectedJob)
		if err != nil {
			if errors.Is(err, terminal.InterruptErr) {
				return 0, nil
			}

			return 0, err
		}
	}

	if selectedJob != "" {
		re := regexp.MustCompile(`(?s)\((.*)\)`)
		m := re.FindAllStringSubmatch(selectedJob, -1)
		return int64(utils.StringToInt(m[0][1])), nil
	} else if len(jobs) > 0 {
		return 0, nil
	}

	pipeline, _, err := opts.Client.Pipelines.GetPipeline(opts.Repo.FullName(), pipelineId)
	if err != nil {
		return 0, err
	}
	// use commit statuses to show external jobs
	cs, _, err := opts.Client.Commits.GetCommitStatuses(opts.Repo.FullName(), pipeline.SHA, &gitlab.GetCommitStatusesOptions{All: gitlab.Ptr(true)})
	if err != nil {
		return 0, nil
	}

	c := opts.IO.Color()

	fmt.Fprint(opts.IO.StdOut, "Getting external jobs...\n")
	for _, status := range cs {
		var s string

		switch status.Status {
		case "success":
			s = c.Green(status.Status)
		case "error":
			s = c.Red(status.Status)
		default:
			s = c.Gray(status.Status)
		}
		fmt.Fprintf(opts.IO.StdOut, "(%s) %s\nURL: %s\n\n", s, c.Bold(status.Name), c.Gray(status.TargetURL))
	}

	fmt.Fprintln(opts.IO.StdErr, "Pipeline has no jobs or external statuses. "+
		"Check for errors in your '.gitlab-ci.yml' and your pipeline configuration.")
	return 0, nil
}

type JobInputs struct {
	JobName            string
	Branch             string
	PipelineId         int
	SelectionPrompt    string
	SelectionPredicate func(s *gitlab.Job) bool
}

type JobOptions struct {
	Client *gitlab.Client
	Repo   glrepo.Interface
	IO     *iostreams.IOStreams
}

func TraceJob(inputs *JobInputs, opts *JobOptions) error {
	jobID, err := GetJobId(inputs, opts)
	if err != nil {
		fmt.Fprintln(opts.IO.StdErr, "invalid job ID:", inputs.JobName)
		return err
	}
	if jobID == 0 {
		return nil
	}
	fmt.Fprintln(opts.IO.StdOut)
	return runTrace(context.Background(), opts.Client, opts.IO.StdOut, opts.Repo.FullName(), jobID)
}

// IDsFromArgs parses list of IDs from space or comma-separated values
func IDsFromArgs(args []string) ([]int, error) {
	var parsedValues []int

	f := func(r rune) bool {
		return r == ',' || r == ' '
	}

	processed := strings.FieldsFunc(strings.Join(args, " "), f)
	for _, v := range processed {
		id, err := strconv.Atoi(v)
		if err != nil {
			return nil, err
		}
		parsedValues = append(parsedValues, id)
	}
	return parsedValues, nil
}
