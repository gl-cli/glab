package api

import (
	"fmt"

	gitlab "gitlab.com/gitlab-org/api/client-go"
)

// GetSchedules returns a scheduled pipeline
// Attention: this is a global variable and may be overridden in tests.
var GetSchedules = func(client *gitlab.Client, l *gitlab.ListPipelineSchedulesOptions, repo string) ([]*gitlab.PipelineSchedule, error) {
	if l.PerPage == 0 {
		l.PerPage = DefaultListLimit
	}

	schedules, _, err := client.PipelineSchedules.ListPipelineSchedules(repo, l)
	if err != nil {
		return nil, err
	}
	return schedules, nil
}

// RunSchedule runs a scheduled pipeline
// Attention: this is a global variable and may be overridden in tests.
var RunSchedule = func(client *gitlab.Client, repo string, schedule int, opts ...gitlab.RequestOptionFunc) error {
	_, err := client.PipelineSchedules.RunPipelineSchedule(repo, schedule, opts...)
	if err != nil {
		return fmt.Errorf("running scheduled pipeline status: %w", err)
	}

	return nil
}
