package api

import (
	"fmt"

	"github.com/xanzy/go-gitlab"
)

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

var RunSchedule = func(client *gitlab.Client, repo string, schedule int, opts ...gitlab.RequestOptionFunc) error {
	if client == nil {
		client = apiClient.Lab()
	}

	_, err := client.PipelineSchedules.RunPipelineSchedule(repo, schedule, opts...)
	if err != nil {
		return fmt.Errorf("running scheduled pipeline status: %w", err)
	}

	return nil
}

var CreateSchedule = func(client *gitlab.Client, repo string, scheduleOpts *gitlab.CreatePipelineScheduleOptions, opts ...gitlab.RequestOptionFunc) (error, *gitlab.PipelineSchedule) {
	if client == nil {
		client = apiClient.Lab()
	}

	schedule, _, err := client.PipelineSchedules.CreatePipelineSchedule(repo, scheduleOpts, opts...)
	if err != nil {
		return fmt.Errorf("creating scheduled pipeline status: %w", err), nil
	}

	return nil, schedule
}

var CreateScheduleVariable = func(client *gitlab.Client, repo string, schedule *gitlab.PipelineSchedule, scheduleVarOpts *gitlab.CreatePipelineScheduleVariableOptions, opts ...gitlab.RequestOptionFunc) error {
	if client == nil {
		client = apiClient.Lab()
	}

	_, _, err := client.PipelineSchedules.CreatePipelineScheduleVariable(repo, schedule.ID, scheduleVarOpts, opts...)
	if err != nil {
		return fmt.Errorf("creating scheduled pipeline status: %w", err)
	}

	return nil
}

var DeleteSchedule = func(client *gitlab.Client, scheduleId int, repo string, opts ...gitlab.RequestOptionFunc) (err error) {
	if client == nil {
		client = apiClient.Lab()
	}

	_, err = client.PipelineSchedules.DeletePipelineSchedule(repo, scheduleId, opts...)
	if err != nil {
		return fmt.Errorf("deleting scheduled pipeline status: %w", err)
	}
	return nil
}
