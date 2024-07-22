package ciutils

import (
	"context"
	"fmt"
	"io"
	"regexp"
	"strconv"
	"sync"
	"time"

	"gitlab.com/gitlab-org/cli/commands/cmdutils"
	"gitlab.com/gitlab-org/cli/internal/glrepo"
	"gitlab.com/gitlab-org/cli/pkg/git"
	"gitlab.com/gitlab-org/cli/pkg/iostreams"
	"gitlab.com/gitlab-org/cli/pkg/prompt"

	"gitlab.com/gitlab-org/cli/api"
	"gitlab.com/gitlab-org/cli/pkg/tableprinter"
	"gitlab.com/gitlab-org/cli/pkg/utils"

	"github.com/AlecAivazis/survey/v2"
	"github.com/AlecAivazis/survey/v2/terminal"
	"github.com/pkg/errors"
	"github.com/xanzy/go-gitlab"
)

var (
	once   sync.Once
	offset int64
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

		for _, pipeline := range p {
			duration := ""

			if pipeline.CreatedAt != nil {
				duration = c.Magenta("(" + utils.TimeToPrettyTimeAgo(*pipeline.CreatedAt) + ")")
			}

			var pipeState string
			if pipeline.Status == "success" {
				pipeState = c.Green(fmt.Sprintf("(%s) • #%s", pipeline.Status, makeHyperlink(s, pipeline)))
			} else if pipeline.Status == "failed" {
				pipeState = c.Red(fmt.Sprintf("(%s) • #%s", pipeline.Status, makeHyperlink(s, pipeline)))
			} else {
				pipeState = c.Gray(fmt.Sprintf("(%s) • #%s", pipeline.Status, makeHyperlink(s, pipeline)))
			}

			table.AddRow(pipeState, fmt.Sprintf("(#%d)", pipeline.IID), pipeline.Ref, duration)
		}

		return table.Render()
	}

	return "No Pipelines available on " + projectID
}

func RunTraceSha(ctx context.Context, apiClient *gitlab.Client, w io.Writer, pid interface{}, sha, name string) error {
	job, err := api.PipelineJobWithSha(apiClient, pid, sha, name)
	if err != nil || job == nil {
		return errors.Wrap(err, "failed to find job")
	}
	return runTrace(ctx, apiClient, w, pid, job.ID)
}

func runTrace(ctx context.Context, apiClient *gitlab.Client, w io.Writer, pid interface{}, jobId int) error {
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

func GetJobId(inputs *JobInputs, opts *JobOptions) (int, error) {
	// If the user hasn't supplied an argument, we display the jobs list interactively.
	if inputs.JobName == "" {
		return getJobIdInteractive(inputs, opts)
	}

	// If the user supplied a job ID, we can use it directly.
	if jobID, err := strconv.Atoi(inputs.JobName); err == nil {
		return jobID, nil
	}

	// Otherwise, we try to find the latest job ID based on the job name.
	pipelineId, err := getPipelineId(inputs, opts)
	if err != nil {
		return 0, fmt.Errorf("get pipeline: %w", err)
	}

	jobs, _, err := opts.ApiClient.Jobs.ListPipelineJobs(opts.Repo.FullName(), pipelineId, nil)
	if err != nil {
		return 0, fmt.Errorf("list pipeline jobs: %w", err)
	}

	for _, job := range jobs {
		if job.Name == inputs.JobName {
			return job.ID, nil
		}
	}

	return 0, fmt.Errorf("pipeline %d contains no jobs.", pipelineId)
}

func getPipelineId(inputs *JobInputs, opts *JobOptions) (int, error) {
	if inputs.PipelineId != 0 {
		return inputs.PipelineId, nil
	}

	branch, err := getBranch(inputs.Branch, opts)
	if err != nil {
		return 0, fmt.Errorf("get branch: %w", err)
	}

	pipeline, err := api.GetLastPipeline(opts.ApiClient, opts.Repo.FullName(), branch)
	if err != nil {
		return 0, fmt.Errorf("get last pipeline: %w", err)
	}
	return pipeline.ID, err
}

func GetDefaultBranch(f *cmdutils.Factory) string {
	repo, err := f.BaseRepo()
	if err != nil {
		return "master"
	}

	remotes, err := f.Remotes()
	if err != nil {
		return "master"
	}

	repoRemote, err := remotes.FindByRepo(repo.RepoOwner(), repo.RepoName())
	if err != nil {
		return "master"
	}

	branch, _ := git.GetDefaultBranch(repoRemote.Name)

	return branch
}

func getBranch(branch string, opts *JobOptions) (string, error) {
	if branch != "" {
		return branch, nil
	}

	branch, err := git.CurrentBranch()
	if err != nil {
		return "", err
	}

	return branch, nil
}

func getJobIdInteractive(inputs *JobInputs, opts *JobOptions) (int, error) {
	pipelineId, err := getPipelineId(inputs, opts)
	if err != nil {
		return 0, err
	}

	fmt.Fprintf(opts.IO.StdOut, "Getting jobs for pipeline %d...\n\n", pipelineId)

	jobs, err := api.GetPipelineJobs(opts.ApiClient, pipelineId, opts.Repo.FullName())
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
		return utils.StringToInt(m[0][1]), nil
	} else if len(jobs) > 0 {
		return 0, nil
	}

	pipeline, err := api.GetPipeline(opts.ApiClient, pipelineId, nil, opts.Repo.FullName())
	if err != nil {
		return 0, err
	}
	// use commit statuses to show external jobs
	cs, err := api.GetCommitStatuses(opts.ApiClient, opts.Repo.FullName(), pipeline.SHA)
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
	ApiClient *gitlab.Client
	Repo      glrepo.Interface
	IO        *iostreams.IOStreams
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
	return runTrace(context.Background(), opts.ApiClient, opts.IO.StdOut, opts.Repo.FullName(), jobID)
}
