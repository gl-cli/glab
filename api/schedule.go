package api

import (
	"fmt"

	gitlab "gitlab.com/gitlab-org/api/client-go"
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

var EditSchedule = func(client *gitlab.Client, repo string, scheduleId int, scheduleOpts *gitlab.EditPipelineScheduleOptions, opts ...gitlab.RequestOptionFunc) (*gitlab.PipelineSchedule, error) {
	if client == nil {
		client = apiClient.Lab()
	}

	schedule, _, err := client.PipelineSchedules.EditPipelineSchedule(repo, scheduleId, scheduleOpts, opts...)
	if err != nil {
		return nil, fmt.Errorf("editing scheduled pipeline status: %w", err)
	}

	return schedule, nil
}

var CreateScheduleVariable = func(client *gitlab.Client, repo string, scheduleId int, scheduleVarOpts *gitlab.CreatePipelineScheduleVariableOptions, opts ...gitlab.RequestOptionFunc) error {
	if client == nil {
		client = apiClient.Lab()
	}

	_, _, err := client.PipelineSchedules.CreatePipelineScheduleVariable(repo, scheduleId, scheduleVarOpts, opts...)
	if err != nil {
		return fmt.Errorf("creating scheduled pipeline status: %w", err)
	}

	return nil
}

var EditScheduleVariable = func(client *gitlab.Client, repo string, scheduleId int, variableKey string, scheduleVarOpts *gitlab.EditPipelineScheduleVariableOptions, opts ...gitlab.RequestOptionFunc) error {
	if client == nil {
		client = apiClient.Lab()
	}

	_, _, err := client.PipelineSchedules.EditPipelineScheduleVariable(repo, scheduleId, variableKey, scheduleVarOpts, opts...)
	if err != nil {
		return fmt.Errorf("editing scheduled pipeline status: %w", err)
	}

	return nil
}

var DeleteScheduleVariable = func(client *gitlab.Client, repo string, scheduleId int, variableKey string, opts ...gitlab.RequestOptionFunc) (err error) {
	if client == nil {
		client = apiClient.Lab()
	}

	_, _, err = client.PipelineSchedules.DeletePipelineScheduleVariable(repo, scheduleId, variableKey, opts...)
	if err != nil {
		return fmt.Errorf("deleting scheduled pipeline status: %w", err)
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
