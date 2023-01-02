package api

import "github.com/xanzy/go-gitlab"

var GetSchedules = func(client *gitlab.Client, l *gitlab.ListPipelineSchedulesOptions, repo string) ([]*gitlab.PipelineSchedule, error) {
	if client == nil {
		client = apiClient.Lab()
	}
	if l.PerPage == 0 {
		l.PerPage = DefaultListLimit
	}

	schedules, _, err := client.PipelineSchedules.ListPipelineSchedules(repo, l)
	if err != nil {
		return nil, err
	}
	return schedules, nil
}
