package export

import (
	"bytes"
	"net/http"
	"testing"

	"github.com/MakeNowJust/heredoc/v2"
	"github.com/google/shlex"
	"github.com/stretchr/testify/assert"
	gitlab "gitlab.com/gitlab-org/api/client-go"
	"gitlab.com/gitlab-org/cli/internal/api"
	"gitlab.com/gitlab-org/cli/internal/config"
	"gitlab.com/gitlab-org/cli/internal/glinstance"
	"gitlab.com/gitlab-org/cli/internal/glrepo"
	"gitlab.com/gitlab-org/cli/internal/testing/cmdtest"
	"gitlab.com/gitlab-org/cli/internal/testing/httpmock"
)

func Test_NewCmdExport(t *testing.T) {
	tests := []struct {
		name     string
		cli      string
		wants    options
		wantsErr bool
	}{
		{
			name:     "no arguments",
			cli:      "",
			wantsErr: false,
		},
		{
			name:     "with group",
			cli:      "--group STH",
			wantsErr: false,
			wants: options{
				group: "STH",
			},
		},
		{
			name:     "missing group",
			cli:      "--group",
			wantsErr: true,
			wants: options{
				group: "STH",
			},
		},
		{
			name:     "with pagination",
			cli:      "--page 11 --per-page 12",
			wantsErr: false,
		},
		{
			name:     "with invalid pagination",
			cli:      "--page aa --per-page bb",
			wantsErr: true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			io, _, _, _ := cmdtest.TestIOStreams()
			f := cmdtest.NewTestFactory(io)

			argv, err := shlex.Split(test.cli)
			assert.NoError(t, err)

			var gotOpts *options
			cmd := NewCmdExport(f, func(opts *options) error {
				gotOpts = opts
				return nil
			})
			cmd.SetArgs(argv)
			cmd.SetIn(&bytes.Buffer{})
			cmd.SetOut(&bytes.Buffer{})
			cmd.SetErr(&bytes.Buffer{})

			_, err = cmd.ExecuteC()
			if test.wantsErr {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)

			assert.Equal(t, test.wants.group, gotOpts.group)
		})
	}
}

func Test_exportRun_project(t *testing.T) {
	reg := &httpmock.Mocker{
		MatchURL: httpmock.FullURL,
	}
	defer reg.Verify(t)

	mockProjectVariables := []gitlab.ProjectVariable{
		{
			Key:              "VAR1",
			Value:            "value1",
			EnvironmentScope: "dev",
		},
		{
			Key:              "VAR2",
			Value:            "value2.1",
			EnvironmentScope: "prod",
		},
		{
			Key:              "VAR2",
			Value:            "value2.2",
			EnvironmentScope: "*",
		},
		{
			Key:              "VAR3",
			Value:            "value3",
			EnvironmentScope: "dev/a",
		},
		{
			Key:              "VAR4",
			Value:            "value4.1",
			EnvironmentScope: "dev/b",
		},
		{
			Key:              "VAR4",
			Value:            "value4.2",
			EnvironmentScope: "feature-1",
		},
		{
			Key:              "VAR4",
			Value:            "value4.3",
			EnvironmentScope: "feature-2",
		},
		{
			Key:              "VAR5",
			Value:            "value5",
			EnvironmentScope: "feature-*",
		},
	}

	io, _, stdout, _ := cmdtest.TestIOStreams()

	tests := []struct {
		scope          string
		format         string
		expectedOutput string
	}{
		{
			scope:  "*",
			format: "json",
			expectedOutput: heredoc.Doc(`
            [
              {
                "key": "VAR1",
                "value": "value1",
                "variable_type": "",
                "protected": false,
                "masked": false,
                "hidden": false,
                "raw": false,
                "environment_scope": "dev",
                "description": ""
              },
              {
                "key": "VAR2",
                "value": "value2.1",
                "variable_type": "",
                "protected": false,
                "masked": false,
                "hidden": false,
                "raw": false,
                "environment_scope": "prod",
                "description": ""
              },
              {
                "key": "VAR2",
                "value": "value2.2",
                "variable_type": "",
                "protected": false,
                "masked": false,
                "hidden": false,
                "raw": false,
                "environment_scope": "*",
                "description": ""
              },
              {
                "key": "VAR3",
                "value": "value3",
                "variable_type": "",
                "protected": false,
                "masked": false,
                "hidden": false,
                "raw": false,
                "environment_scope": "dev/a",
                "description": ""
              },
              {
                "key": "VAR4",
                "value": "value4.1",
                "variable_type": "",
                "protected": false,
                "masked": false,
                "hidden": false,
                "raw": false,
                "environment_scope": "dev/b",
                "description": ""
              },
              {
                "key": "VAR4",
                "value": "value4.2",
                "variable_type": "",
                "protected": false,
                "masked": false,
                "hidden": false,
                "raw": false,
                "environment_scope": "feature-1",
                "description": ""
              },
              {
                "key": "VAR4",
                "value": "value4.3",
                "variable_type": "",
                "protected": false,
                "masked": false,
                "hidden": false,
                "raw": false,
                "environment_scope": "feature-2",
                "description": ""
              },
              {
                "key": "VAR5",
                "value": "value5",
                "variable_type": "",
                "protected": false,
                "masked": false,
                "hidden": false,
                "raw": false,
                "environment_scope": "feature-*",
                "description": ""
              }
            ]
            `),
		},
		{
			scope:  "dev/b",
			format: "json",
			expectedOutput: heredoc.Doc(`
            [
              {
                "key": "VAR2",
                "value": "value2.2",
                "variable_type": "",
                "protected": false,
                "masked": false,
                "hidden": false,
                "raw": false,
                "environment_scope": "*",
                "description": ""
              },
              {
                "key": "VAR4",
                "value": "value4.1",
                "variable_type": "",
                "protected": false,
                "masked": false,
                "hidden": false,
                "raw": false,
                "environment_scope": "dev/b",
                "description": ""
              }
            ]
            `),
		},
		{
			scope:          "*",
			format:         "env",
			expectedOutput: "VAR1=\"value1\"\nVAR2=\"value2.1\"\nVAR3=\"value3\"\nVAR4=\"value4.1\"\nVAR4=\"value4.2\"\nVAR4=\"value4.3\"\nVAR5=\"value5\"\n",
		},
		{
			scope:          "*",
			format:         "export",
			expectedOutput: "export VAR1=\"value1\"\nexport VAR2=\"value2.1\"\nexport VAR3=\"value3\"\nexport VAR4=\"value4.1\"\nexport VAR4=\"value4.2\"\nexport VAR4=\"value4.3\"\nexport VAR5=\"value5\"\n",
		},
		{
			scope:          "dev",
			format:         "env",
			expectedOutput: "VAR1=\"value1\"\nVAR2=\"value2.2\"\n",
		},
		{
			scope:          "dev",
			format:         "export",
			expectedOutput: "export VAR1=\"value1\"\nexport VAR2=\"value2.2\"\n",
		},
		{
			scope:          "prod",
			format:         "env",
			expectedOutput: "VAR2=\"value2.1\"\n",
		},
		{
			scope:          "prod",
			format:         "export",
			expectedOutput: "export VAR2=\"value2.1\"\n",
		},
		{
			scope:          "dev/a",
			format:         "env",
			expectedOutput: "VAR3=\"value3\"\nVAR2=\"value2.2\"\n",
		},
		{
			scope:          "dev/a",
			format:         "export",
			expectedOutput: "export VAR3=\"value3\"\nexport VAR2=\"value2.2\"\n",
		},
		{
			scope:          "feature-1",
			format:         "env",
			expectedOutput: "VAR4=\"value4.2\"\nVAR2=\"value2.2\"\nVAR5=\"value5\"\n",
		},
		{
			scope:          "feature-1",
			format:         "export",
			expectedOutput: "export VAR4=\"value4.2\"\nexport VAR2=\"value2.2\"\nexport VAR5=\"value5\"\n",
		},
		{
			scope:          "feature-2",
			format:         "env",
			expectedOutput: "VAR4=\"value4.3\"\nVAR2=\"value2.2\"\nVAR5=\"value5\"\n",
		},
		{
			scope:          "feature-2",
			format:         "export",
			expectedOutput: "export VAR4=\"value4.3\"\nexport VAR2=\"value2.2\"\nexport VAR5=\"value5\"\n",
		},
	}

	for _, test := range tests {
		t.Run(test.scope+"_"+test.format, func(t *testing.T) {
			reg.RegisterResponder(http.MethodGet, "https://gitlab.com/api/v4/projects/owner%2Frepo/variables?page=1&per_page=10",
				httpmock.NewJSONResponse(http.StatusOK, mockProjectVariables),
			)
			opts := &options{
				apiClient: func(repoHost string, cfg config.Config) (*api.Client, error) {
					return cmdtest.NewTestApiClient(t, &http.Client{Transport: reg}, "", "gitlab.com"), nil
				},
				baseRepo: func() (glrepo.Interface, error) {
					return glrepo.FromFullName("owner/repo", glinstance.DefaultHostname)
				},
				config:       config.NewBlankConfig(),
				io:           io,
				page:         1,
				perPage:      10,
				outputFormat: test.format,
				scope:        test.scope,
			}

			err := opts.run()
			assert.NoError(t, err)
			assert.Equal(t, test.expectedOutput, stdout.String())
			stdout.Reset()
		})
	}
}

func Test_exportRun_group(t *testing.T) {
	reg := &httpmock.Mocker{
		MatchURL: httpmock.FullURL,
	}
	defer reg.Verify(t)

	mockGroupVariables := []gitlab.GroupVariable{
		{
			Key:              "VAR1",
			Value:            "\"value1\"",
			EnvironmentScope: "dev",
		},
		{
			Key:              "VAR2",
			Value:            "value2.1",
			EnvironmentScope: "prod",
		},
		{
			Key:              "VAR2",
			Value:            "value2.2",
			EnvironmentScope: "*",
		},
		{
			Key:              "VAR3",
			Value:            "value3",
			EnvironmentScope: "dev/a",
		},
		{
			Key:              "VAR4",
			Value:            "value4.1",
			EnvironmentScope: "dev/b",
		},
		{
			Key:              "VAR4",
			Value:            "value4.2",
			EnvironmentScope: "feature-1",
		},
		{
			Key:              "VAR4",
			Value:            "value4.3",
			EnvironmentScope: "feature-2",
		},
		{
			Key:              "VAR5",
			Value:            "value5",
			EnvironmentScope: "feature-*",
		},
	}

	io, _, stdout, _ := cmdtest.TestIOStreams()

	tests := []struct {
		scope          string
		format         string
		expectedOutput string
	}{
		{
			scope:  "*",
			format: "json",
			expectedOutput: heredoc.Doc(`
            [
              {
                "key": "VAR1",
                "value": "\"value1\"",
                "variable_type": "",
                "protected": false,
                "masked": false,
                "hidden": false,
                "raw": false,
                "environment_scope": "dev",
                "description": ""
              },
              {
                "key": "VAR2",
                "value": "value2.1",
                "variable_type": "",
                "protected": false,
                "masked": false,
                "hidden": false,
                "raw": false,
                "environment_scope": "prod",
                "description": ""
              },
              {
                "key": "VAR2",
                "value": "value2.2",
                "variable_type": "",
                "protected": false,
                "masked": false,
                "hidden": false,
                "raw": false,
                "environment_scope": "*",
                "description": ""
              },
              {
                "key": "VAR3",
                "value": "value3",
                "variable_type": "",
                "protected": false,
                "masked": false,
                "hidden": false,
                "raw": false,
                "environment_scope": "dev/a",
                "description": ""
              },
              {
                "key": "VAR4",
                "value": "value4.1",
                "variable_type": "",
                "protected": false,
                "masked": false,
                "hidden": false,
                "raw": false,
                "environment_scope": "dev/b",
                "description": ""
              },
              {
                "key": "VAR4",
                "value": "value4.2",
                "variable_type": "",
                "protected": false,
                "masked": false,
                "hidden": false,
                "raw": false,
                "environment_scope": "feature-1",
                "description": ""
              },
              {
                "key": "VAR4",
                "value": "value4.3",
                "variable_type": "",
                "protected": false,
                "masked": false,
                "hidden": false,
                "raw": false,
                "environment_scope": "feature-2",
                "description": ""
              },
              {
                "key": "VAR5",
                "value": "value5",
                "variable_type": "",
                "protected": false,
                "masked": false,
                "hidden": false,
                "raw": false,
                "environment_scope": "feature-*",
                "description": ""
              }
            ]
            `),
		},
		{
			scope:  "dev/b",
			format: "json",
			expectedOutput: heredoc.Doc(`
            [
              {
                "key": "VAR2",
                "value": "value2.2",
                "variable_type": "",
                "protected": false,
                "masked": false,
                "hidden": false,
                "raw": false,
                "environment_scope": "*",
                "description": ""
              },
              {
                "key": "VAR4",
                "value": "value4.1",
                "variable_type": "",
                "protected": false,
                "masked": false,
                "hidden": false,
                "raw": false,
                "environment_scope": "dev/b",
                "description": ""
              }
            ]
            `),
		},
		{
			scope:          "*",
			format:         "env",
			expectedOutput: "VAR1=\"value1\"\nVAR2=value2.1\nVAR3=value3\nVAR4=value4.1\nVAR4=value4.2\nVAR4=value4.3\nVAR5=value5\n",
		},
		{
			scope:          "*",
			format:         "export",
			expectedOutput: "export VAR1=\"value1\"\nexport VAR2=value2.1\nexport VAR3=value3\nexport VAR4=value4.1\nexport VAR4=value4.2\nexport VAR4=value4.3\nexport VAR5=value5\n",
		},
		{
			scope:          "dev",
			format:         "env",
			expectedOutput: "VAR1=\"value1\"\nVAR2=value2.2\n",
		},
		{
			scope:          "dev",
			format:         "export",
			expectedOutput: "export VAR1=\"value1\"\nexport VAR2=value2.2\n",
		},
		{
			scope:          "prod",
			format:         "env",
			expectedOutput: "VAR2=value2.1\n",
		},
		{
			scope:          "prod",
			format:         "export",
			expectedOutput: "export VAR2=value2.1\n",
		},
		{
			scope:          "dev/a",
			format:         "env",
			expectedOutput: "VAR3=value3\nVAR2=value2.2\n",
		},
		{
			scope:          "dev/a",
			format:         "export",
			expectedOutput: "export VAR3=value3\nexport VAR2=value2.2\n",
		},
		{
			scope:          "feature-1",
			format:         "env",
			expectedOutput: "VAR4=value4.2\nVAR2=value2.2\nVAR5=value5\n",
		},
		{
			scope:          "feature-1",
			format:         "export",
			expectedOutput: "export VAR4=value4.2\nexport VAR2=value2.2\nexport VAR5=value5\n",
		},
		{
			scope:          "feature-2",
			format:         "env",
			expectedOutput: "VAR4=value4.3\nVAR2=value2.2\nVAR5=value5\n",
		},
		{
			scope:          "feature-2",
			format:         "export",
			expectedOutput: "export VAR4=value4.3\nexport VAR2=value2.2\nexport VAR5=value5\n",
		},
	}

	for _, test := range tests {
		t.Run(test.scope+"_"+test.format, func(t *testing.T) {
			reg.RegisterResponder(http.MethodGet, "https://gitlab.com/api/v4/groups/group/variables?page=1&per_page=10",
				httpmock.NewJSONResponse(http.StatusOK, mockGroupVariables),
			)
			opts := &options{
				apiClient: func(repoHost string, cfg config.Config) (*api.Client, error) {
					return cmdtest.NewTestApiClient(t, &http.Client{Transport: reg}, "", "gitlab.com"), nil
				},
				baseRepo: func() (glrepo.Interface, error) {
					return glrepo.FromFullName("owner/repo", glinstance.DefaultHostname)
				},
				config:       config.NewBlankConfig(),
				io:           io,
				page:         1,
				perPage:      10,
				outputFormat: test.format,
				scope:        test.scope,
				group:        "group",
			}

			err := opts.run()
			assert.NoError(t, err)
			assert.Equal(t, test.expectedOutput, stdout.String())
			stdout.Reset()
		})
	}
}
