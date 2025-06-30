package main

import (
	"strconv"
	"strings"

	"github.com/spf13/cobra"
	gitlab "gitlab.com/gitlab-org/api/client-go"

	"gitlab.com/gitlab-org/cli/internal/cmdutils"
	"gitlab.com/gitlab-org/cli/internal/config"
	"gitlab.com/gitlab-org/cli/internal/dbg"
)

func addTelemetryHook(f cmdutils.Factory, cmd *cobra.Command) func() {
	return func() {
		go sendTelemetryData(f, cmd)
	}
}

// isTelemetryEnabled checks if usage data is disabled via config or env var
func isTelemetryEnabled(cfg config.Config) bool {
	telemetryEnabled, _ := cfg.Get("", "telemetry")
	enabled, err := strconv.ParseBool(telemetryEnabled)
	if err != nil {
		dbg.Debugf("Could not parse telemetry config value %s - defaulting to 'true'", telemetryEnabled)
		return true
	}

	return enabled
}

// parseCommand parses a command string and returns components
func parseCommand(parts []string) (string, string, string) {
	if len(parts) < 2 {
		return "", "", ""
	}

	// glab is always the first value, command is the next
	command := parts[1]

	subcommandParts := parts[2:]
	subcommand := strings.Join(subcommandParts, " ")

	fullCommand := command
	if subcommand != "" {
		fullCommand += " " + subcommand
	}

	return command, subcommand, fullCommand
}

func sendTelemetryData(f cmdutils.Factory, cmd *cobra.Command) {
	var projectID int
	var namespaceID int

	if cmd == nil {
		return
	}

	unparsedCommand := strings.Split(cmd.CommandPath(), " ")

	command, subcommand, fullCommand := parseCommand(unparsedCommand)

	var client *gitlab.Client
	repo, err := f.BaseRepo()
	if err != nil {
		dbg.Debug("Could not determine base repo in telemetry hook: ", err.Error())

		c, err := f.ApiClient("", f.Config())
		if err != nil {
			f.IO().Logf("Could not get API Client in telemetry hook: %s", err.Error())
			return
		}
		client = c.Lab()
	} else {
		c, err := f.HttpClient()
		if err != nil {
			f.IO().Logf("Could not get API Client in telemetry hook: %s", err.Error())
			return
		}
		client = c

		project, err := repo.Project(client)
		if err == nil {
			projectID = project.ID
			namespaceID = project.Namespace.ID
		}
	}

	_, err = client.UsageData.TrackEvent(&gitlab.TrackEventOptions{
		Event:          "gitlab_cli_command_used",
		NamespaceID:    gitlab.Ptr(namespaceID),
		ProjectID:      gitlab.Ptr(projectID),
		SendToSnowplow: gitlab.Ptr(true),
		AdditionalProperties: map[string]string{
			"label":                  command,
			"property":               subcommand,
			"command_and_subcommand": fullCommand,
		},
	})
	if err != nil {
		f.IO().Logf("Could not send telemetry data: %s", err.Error())
	}
}
