package main

import (
	"strconv"
	"strings"

	"github.com/spf13/cobra"

	gitlab "gitlab.com/gitlab-org/api/client-go"

	"gitlab.com/gitlab-org/cli/internal/cmdutils"
	"gitlab.com/gitlab-org/cli/internal/config"
	"gitlab.com/gitlab-org/cli/internal/dbg"
	"gitlab.com/gitlab-org/cli/internal/utils"
)

func addTelemetryHook(f cmdutils.Factory, cmd *cobra.Command) func() {
	return func() {
		go sendTelemetryData(f, cmd)
	}
}

// isTelemetryEnabled checks if usage data is disabled via config or env var
func isTelemetryEnabled(cfg config.Config) bool {
	if enabled, found := utils.IsEnvVarEnabled("GLAB_SEND_TELEMETRY"); found {
		return enabled
	}

	// Fall back to config value if env var not set
	if telemetryEnabled, _ := cfg.Get("", "telemetry"); telemetryEnabled != "" {
		if telemetryEnabledParsed, err := strconv.ParseBool(telemetryEnabled); err == nil {
			return telemetryEnabledParsed
		}
	}

	return true
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
	var projectID int64
	var namespaceID int64

	if cmd == nil {
		return
	}

	unparsedCommand := strings.Split(cmd.CommandPath(), " ")

	command, subcommand, fullCommand := parseCommand(unparsedCommand)

	var client *gitlab.Client
	repo, err := f.BaseRepo()
	if err != nil {
		dbg.Debug("Could not determine base repo in telemetry hook: ", err.Error())

		c, err := f.ApiClient("")
		if err != nil {
			f.IO().LogErrorf("Could not get API Client in telemetry hook: %s", err.Error())
			return
		}
		client = c.Lab()
	} else {
		c, err := f.GitLabClient()
		if err != nil {
			f.IO().LogErrorf("Could not get API Client in telemetry hook: %s", err.Error())
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
		f.IO().LogErrorf("Could not send telemetry data: %s", err.Error())
	}
}
