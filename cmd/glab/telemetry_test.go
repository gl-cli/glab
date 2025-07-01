package main

import (
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/require"
	gitlab "gitlab.com/gitlab-org/api/client-go"
	gitlab_testing "gitlab.com/gitlab-org/api/client-go/testing"
	"go.uber.org/mock/gomock"

	"gitlab.com/gitlab-org/cli/internal/config"
	"gitlab.com/gitlab-org/cli/internal/testing/cmdtest"
)

func Test_sendTelemetryData(t *testing.T) {
	tests := []struct {
		name        string
		args        []string
		cobraMocks  []*cobra.Command
		command     string
		subcommand  string
		fullCommand string
	}{
		{
			name: "command with subcommand",
			cobraMocks: []*cobra.Command{
				{Use: "glab"},
				{Use: "mr"},
				{Use: "view"},
			},
			command:     "mr",
			subcommand:  "view",
			fullCommand: "mr view",
		},
		{
			name: "command with multiple subcommands",
			cobraMocks: []*cobra.Command{
				{Use: "glab"},
				{Use: "command"},
				{Use: "subcommand1"},
				{Use: "subcommand2"},
			},
			args:        []string{"glab", "command", "subcommand1", "subcommand2"},
			command:     "command",
			subcommand:  "subcommand1 subcommand2",
			fullCommand: "command subcommand1 subcommand2",
		},
		{
			name: "single command only",
			cobraMocks: []*cobra.Command{
				{Use: "glab"},
				{Use: "version"},
			},
			command:     "version",
			subcommand:  "",
			fullCommand: "version",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tc := gitlab_testing.NewTestClient(t)
			ios, _, _, _ := cmdtest.TestIOStreams(cmdtest.WithTestIOStreamsAsTTY(true))

			f := cmdtest.NewTestFactory(ios,
				cmdtest.WithGitLabClient(tc.Client),
			)

			project := gitlab.Project{
				ID:        123,
				Namespace: &gitlab.ProjectNamespace{ID: 123},
			}

			tc.MockProjects.EXPECT().
				GetProject(gomock.Any(), gomock.Any(), gomock.Any()).
				Return(&project, &gitlab.Response{}, nil)

			tc.MockUsageData.EXPECT().
				TrackEvent(&gitlab.TrackEventOptions{
					Event:          "gitlab_cli_command_used",
					NamespaceID:    gitlab.Ptr(project.Namespace.ID),
					ProjectID:      gitlab.Ptr(project.ID),
					SendToSnowplow: gitlab.Ptr(true),
					AdditionalProperties: map[string]string{
						"label":                  tt.command,
						"property":               tt.subcommand,
						"command_and_subcommand": tt.fullCommand,
					},
				})

			passedCommand := tt.cobraMocks[0]
			numberOfCommands := len(tt.cobraMocks)

			for i, cmd := range tt.cobraMocks {
				if i < numberOfCommands && i > 0 {
					tt.cobraMocks[i-1].AddCommand(cmd)

					passedCommand = cmd
				}
			}

			sendTelemetryData(f, passedCommand)
		})
	}
}

func Test_parseCommand(t *testing.T) {
	tests := []struct {
		name        string
		cmdString   []string
		command     string
		subcommand  string
		fullCommand string
	}{
		{
			name:        "basic command",
			cmdString:   []string{"glab", "mr", "list"},
			command:     "mr",
			subcommand:  "list",
			fullCommand: "mr list",
		},
		{
			name:        "multiple subcommands",
			cmdString:   []string{"glab", "command", "subcommand1", "subcommand2", "subcommand3"},
			command:     "command",
			subcommand:  "subcommand1 subcommand2 subcommand3",
			fullCommand: "command subcommand1 subcommand2 subcommand3",
		},
		{
			name:        "no subcommand",
			cmdString:   []string{"glab", "mr"},
			command:     "mr",
			subcommand:  "",
			fullCommand: "mr",
		},
		{
			name:        "too short of a command",
			cmdString:   []string{"glab"},
			command:     "",
			subcommand:  "",
			fullCommand: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			command, subcommand, fullCommand := parseCommand(tt.cmdString)

			require := require.New(t)
			require.Equal(tt.command, command)
			require.Equal(tt.subcommand, subcommand)
			require.Equal(tt.fullCommand, fullCommand)
		})
	}
}

func TestIsTelemetryEnabled(t *testing.T) {
	tests := []struct {
		name           string
		configYaml     string
		expectedResult bool
	}{
		{
			name:           "enabled with 'true' value",
			configYaml:     "telemetry: true",
			expectedResult: true,
		},
		{
			name:           "enabled with '1' value",
			configYaml:     "telemetry: '1'",
			expectedResult: true,
		},
		{
			name:           "disabled with 'false' value",
			configYaml:     "telemetry: false",
			expectedResult: false,
		},
		{
			name:           "disabled with '0' value",
			configYaml:     "telemetry: '0'",
			expectedResult: false,
		},
		{
			name:           "enabled with empty value",
			configYaml:     "telemetry: ''",
			expectedResult: true,
		},
		{
			name:           "enabled with other value",
			configYaml:     "telemetry: something",
			expectedResult: true,
		},
		{
			name:           "no config value set",
			expectedResult: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			restore := config.StubConfig(tt.configYaml, "")
			defer restore()

			cfg, err := config.ParseConfig("config.yml")
			if tt.configYaml == "" {
				cfg = config.NewBlankConfig()
			} else {
				require.NoError(t, err)
			}

			result := isTelemetryEnabled(cfg)

			require.Equal(t, tt.expectedResult, result)
		})
	}
}
