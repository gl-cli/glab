package api

import (
	"bytes"
	"io"
	"sort"

	gitlab "gitlab.com/gitlab-org/api/client-go"
	"gitlab.com/gitlab-org/cli/pkg/git"
)

var RetryPipeline = func(client *gitlab.Client, pid int, repo string) (*gitlab.Pipeline, error) {
	pipe, _, err := client.Pipelines.RetryPipelineBuild(repo, pid)
	if err != nil {
		return nil, err
	}
	return pipe, nil
}

var PlayPipelineJob = func(client *gitlab.Client, pid int, repo string) (*gitlab.Job, error) {
	playOptions := gitlab.PlayJobOptions{}
	pipe, _, err := client.Jobs.PlayJob(repo, pid, &playOptions)
	if err != nil {
		return nil, err
	}
	return pipe, nil
}

var RetryPipelineJob = func(client *gitlab.Client, pid int, repo string) (*gitlab.Job, error) {
	pipe, _, err := client.Jobs.RetryJob(repo, pid)
	if err != nil {
		return nil, err
	}
	return pipe, nil
}

var CancelPipelineJob = func(client *gitlab.Client, repo string, jobID int) (*gitlab.Job, error) {
	pipe, _, err := client.Jobs.CancelJob(repo, jobID)
	if err != nil {
		return nil, err
	}
	return pipe, nil
}

var PlayOrRetryJobs = func(client *gitlab.Client, repo string, jobID int, status string) (*gitlab.Job, error) {
	switch status {
	case "pending", "running":
		return nil, nil
	case "manual":
		j, err := PlayPipelineJob(client, jobID, repo)
		if err != nil {
			return nil, err
		}
		return j, nil
	default:

		j, err := RetryPipelineJob(client, jobID, repo)
		if err != nil {
			return nil, err
		}

		return j, nil
	}
}

var ErasePipelineJob = func(client *gitlab.Client, pid int, repo string) (*gitlab.Job, error) {
	pipe, _, err := client.Jobs.EraseJob(repo, pid)
	if err != nil {
		return nil, err
	}
	return pipe, nil
}

var GetPipelineJob = func(client *gitlab.Client, jid int, repo string) (*gitlab.Job, error) {
	job, _, err := client.Jobs.GetJob(repo, jid)
	return job, err
}

var GetJobs = func(client *gitlab.Client, repo string, opts *gitlab.ListJobsOptions) ([]*gitlab.Job, error) {
	if opts == nil {
		opts = &gitlab.ListJobsOptions{}
	}
	jobs, _, err := client.Jobs.ListProjectJobs(repo, opts)
	if err != nil {
		return nil, err
	}
	return jobs, nil
}

var GetLatestPipeline = func(client *gitlab.Client, repo string, ref string) (*gitlab.Pipeline, error) {
	l := &gitlab.GetLatestPipelineOptions{
		Ref: gitlab.Ptr(ref),
	}

	pipeline, _, err := client.Pipelines.GetLatestPipeline(repo, l)
	if err != nil {
		return nil, err
	}

	return pipeline, nil
}

var GetPipelines = func(client *gitlab.Client, l *gitlab.ListProjectPipelinesOptions, repo any) ([]*gitlab.PipelineInfo, error) {
	if l.PerPage == 0 {
		l.PerPage = DefaultListLimit
	}

	pipes, _, err := client.Pipelines.ListProjectPipelines(repo, l)
	if err != nil {
		return nil, err
	}
	return pipes, nil
}

var GetPipeline = func(client *gitlab.Client, pid int, l *gitlab.RequestOptionFunc, repo any) (*gitlab.Pipeline, error) {
	pipe, _, err := client.Pipelines.GetPipeline(repo, pid)
	if err != nil {
		return nil, err
	}
	return pipe, nil
}

var GetPipelineVariables = func(client *gitlab.Client, pid int, l *gitlab.RequestOptionFunc, projectID int) ([]*gitlab.PipelineVariable, error) {
	pipelineVars, _, err := client.Pipelines.GetPipelineVariables(projectID, pid)
	if err != nil {
		return nil, err
	}
	return pipelineVars, nil
}

var GetPipelineJobs = func(client *gitlab.Client, pid int, repo string) ([]*gitlab.Job, error) {
	pipeJobs := make([]*gitlab.Job, 0, 10)
	listOptions := &gitlab.ListJobsOptions{
		ListOptions: gitlab.ListOptions{
			PerPage: 100,
		},
	}
	for {
		pageJobs, resp, err := client.Jobs.ListPipelineJobs(repo, pid, listOptions)
		if err != nil {
			return nil, err
		}
		pipeJobs = append(pipeJobs, pageJobs...)
		if resp.CurrentPage == resp.TotalPages {
			break
		}
		listOptions.Page = resp.NextPage
		if resp.CurrentPage >= resp.TotalPages {
			break
		}
	}
	return pipeJobs, nil
}

var GetPipelineJobLog = func(client *gitlab.Client, jobID int, repo string) (io.Reader, error) {
	pipeJoblog, _, err := client.Jobs.GetTraceFile(repo, jobID)
	if err != nil {
		return nil, err
	}
	return pipeJoblog, nil
}

var GetSinglePipeline = func(client *gitlab.Client, pid int, repo string) (*gitlab.Pipeline, error) {
	pipes, _, err := client.Pipelines.GetPipeline(repo, pid)
	if err != nil {
		return nil, err
	}
	return pipes, nil
}

var GetCommit = func(client *gitlab.Client, repo string, ref string) (*gitlab.Commit, error) {
	c, _, err := client.Commits.GetCommit(repo, ref, nil)
	if err != nil {
		return nil, err
	}
	return c, nil
}

var GetPipelineFromBranch = func(client *gitlab.Client, ref, repo string) ([]*gitlab.Job, error) {
	var err error
	if ref == "" {
		ref, err = git.CurrentBranch()
		if err != nil {
			return nil, err
		}
	}

	pipeline, err := GetLatestPipeline(client, repo, ref)
	if err != nil {
		return nil, err
	}
	jobs, err := GetPipelineJobs(client, pipeline.ID, repo)
	if err != nil {
		return nil, err
	}
	return jobs, nil
}

var PipelineJobWithSha = func(client *gitlab.Client, pid any, sha, name string) (*gitlab.Job, error) {
	jobs, _, err := PipelineJobsWithSha(client, pid, sha)
	if len(jobs) == 0 || err != nil {
		return nil, err
	}
	var (
		job          *gitlab.Job
		lastRunning  *gitlab.Job
		firstPending *gitlab.Job
	)

	for _, j := range jobs {
		if j.Status == "running" {
			lastRunning = j
		}
		if j.Status == "pending" && firstPending == nil {
			firstPending = j
		}
		if j.Name == name {
			job = j
			// don't break because there may be a newer version of the job
		}
	}
	if job == nil {
		job = lastRunning
	}
	if job == nil {
		job = firstPending
	}
	if job == nil {
		job = jobs[len(jobs)-1]
	}
	return job, err
}

type JobSort struct {
	Jobs []*gitlab.Job
}

func (s JobSort) Len() int      { return len(s.Jobs) }
func (s JobSort) Swap(i, j int) { s.Jobs[i], s.Jobs[j] = s.Jobs[j], s.Jobs[i] }
func (s JobSort) Less(i, j int) bool {
	return (*s.Jobs[i].CreatedAt).Before(*s.Jobs[j].CreatedAt)
}

type BridgeSort struct {
	Bridges []*gitlab.Bridge
}

func (s BridgeSort) Len() int      { return len(s.Bridges) }
func (s BridgeSort) Swap(i, j int) { s.Bridges[i], s.Bridges[j] = s.Bridges[j], s.Bridges[i] }
func (s BridgeSort) Less(i, j int) bool {
	return (*s.Bridges[i].CreatedAt).Before(*s.Bridges[j].CreatedAt)
}

// PipelineJobsWithID returns a list of jobs in a pipeline for a id.
// The jobs are returned in the order in which they were created
var PipelineJobsWithID = func(client *gitlab.Client, pid any, ppid int) ([]*gitlab.Job, []*gitlab.Bridge, error) {
	opts := &gitlab.ListJobsOptions{
		ListOptions: gitlab.ListOptions{
			PerPage: 500,
		},
	}
	jobsList := make([]*gitlab.Job, 0)
	for {
		jobs, resp, err := client.Jobs.ListPipelineJobs(pid, ppid, opts)
		if err != nil {
			return nil, nil, err
		}
		opts.Page = resp.NextPage
		jobsList = append(jobsList, jobs...)
		if resp.CurrentPage == resp.TotalPages {
			break
		}
	}
	opts.Page = 0
	bridgesList := make([]*gitlab.Bridge, 0)
	for {
		bridges, resp, err := client.Jobs.ListPipelineBridges(pid, ppid, opts)
		if err != nil {
			return nil, nil, err
		}
		opts.Page = resp.NextPage
		bridgesList = append(bridgesList, bridges...)
		if resp.CurrentPage == resp.TotalPages {
			break
		}
	}
	// ListPipelineJobs returns jobs sorted by ID in descending order instead of returning
	// them in the order they were created, so we restore the order using the createdAt
	sort.Sort(JobSort{Jobs: jobsList})
	sort.Sort(BridgeSort{Bridges: bridgesList})
	return jobsList, bridgesList, nil
}

// PipelineJobsWithSha returns a list of jobs in a pipeline for a given commit sha.
// The jobs are returned in the order in which they were created
var PipelineJobsWithSha = func(client *gitlab.Client, pid any, sha string) ([]*gitlab.Job, []*gitlab.Bridge, error) {
	pipelines, err := GetPipelines(client, &gitlab.ListProjectPipelinesOptions{
		SHA: gitlab.Ptr(sha),
	}, pid)
	if len(pipelines) == 0 || err != nil {
		return nil, nil, err
	}
	return PipelineJobsWithID(client, pid, pipelines[0].ID)
}

var ProjectNamespaceLint = func(client *gitlab.Client, projectID int, content string, ref string, dryRun bool, includeJobs bool) (*gitlab.ProjectLintResult, error) {
	c, _, err := client.Validate.ProjectNamespaceLint(
		projectID,
		&gitlab.ProjectNamespaceLintOptions{
			Content:     &content,
			DryRun:      &dryRun,
			Ref:         &ref,
			IncludeJobs: &includeJobs,
		},
	)
	if err != nil {
		return nil, err
	}
	return c, nil
}

var DeletePipeline = func(client *gitlab.Client, projectID any, pipeID int) error {
	_, err := client.Pipelines.DeletePipeline(projectID, pipeID)
	if err != nil {
		return err
	}
	return nil
}

var ListProjectPipelines = func(client *gitlab.Client, projectID any, opts *gitlab.ListProjectPipelinesOptions) ([]*gitlab.PipelineInfo, error) {
	pipes, _, err := client.Pipelines.ListProjectPipelines(projectID, opts)
	if err != nil {
		return pipes, err
	}
	return pipes, nil
}

var CreatePipeline = func(client *gitlab.Client, projectID any, opts *gitlab.CreatePipelineOptions) (*gitlab.Pipeline, error) {
	pipe, _, err := client.Pipelines.CreatePipeline(projectID, opts)
	return pipe, err
}

var CreateMergeRequestPipeline = func(client *gitlab.Client, projectID interface{}, mr int) (*gitlab.PipelineInfo, error) {
	pipe, _, err := client.MergeRequests.CreateMergeRequestPipeline(projectID, mr)
	return pipe, err
}

var RunPipelineTrigger = func(client *gitlab.Client, projectID any, opts *gitlab.RunPipelineTriggerOptions) (*gitlab.Pipeline, error) {
	pipe, _, err := client.PipelineTriggers.RunPipelineTrigger(projectID, opts)
	return pipe, err
}

var DownloadArtifactJob = func(client *gitlab.Client, repo string, ref string, opts *gitlab.DownloadArtifactsFileOptions) (*bytes.Reader, error) {
	if opts == nil {
		opts = &gitlab.DownloadArtifactsFileOptions{}
	}
	jobs, _, err := client.Jobs.DownloadArtifactsFile(repo, ref, opts, nil)
	if err != nil {
		return nil, err
	}
	return jobs, nil
}
