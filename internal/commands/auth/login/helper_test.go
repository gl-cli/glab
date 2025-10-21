package login

import (
	"fmt"
	"strings"
	"testing"

	"gitlab.com/gitlab-org/cli/internal/config"
	"gitlab.com/gitlab-org/cli/internal/testing/cmdtest"

	"github.com/MakeNowJust/heredoc/v2"
	"github.com/stretchr/testify/assert"
)

func Test_helperRun(t *testing.T) {
	// NOTE: we have to remove the values for the possible token env variables, because it'll take precedence over the config.
	// See config.EnvKeyEquivalence function.
	t.Setenv("GITLAB_TOKEN", "")
	t.Setenv("GITLAB_ACCESS_TOKEN", "")
	t.Setenv("OAUTH_TOKEN", "")
	t.Setenv("USER", "")

	tests := []struct {
		name            string
		opts            options
		input           string
		wantStdout      []string
		wantStderr      string
		wantErr         bool
		wantValidateErr bool
	}{
		{
			name: "host only, credentials found",
			opts: options{
				operation: "get",
				config: func() config.Config {
					return config.NewFromString(heredoc.Doc(`
						_source: "/Users/monalisa/.config/glab/config.yml"
						hosts:
						  example.com:
						    user: "monalisa"
						    token: "some-password"
					`))
				},
			},
			input: heredoc.Doc(`
				protocol=https
				host=example.com
			`),
			wantErr:         false,
			wantValidateErr: false,
			wantStdout: []string{
				"username=monalisa",
				"password=some-password",
			},
			wantStderr: "",
		},
		{
			name: "host plus user",
			opts: options{
				operation: "get",
				config: func() config.Config {
					return config.NewFromString(heredoc.Doc(`
						_source: "/Users/monalisa/.config/glab/config.yml"
						hosts:
						  example.com:
						    user: "monalisa"
						    token: "some-password"
					`))
				},
			},
			input: heredoc.Doc(`
				protocol=https
				host=example.com
				username=monalisa
			`),
			wantErr:         false,
			wantValidateErr: false,
			wantStdout: []string{
				"username=monalisa",
				"password=some-password",
			},
			wantStderr: "",
		},
		{
			name: "url input",
			opts: options{
				operation: "get",
				config: func() config.Config {
					return config.NewFromString(heredoc.Doc(`
						_source: "/Users/monalisa/.config/glab/config.yml"
						hosts:
						  example.com:
						    user: "monalisa"
						    token: "some-password"
					`))
				},
			},
			input: heredoc.Doc(`
				url=https://monalisa@example.com
			`),
			wantErr:         false,
			wantValidateErr: false,
			wantStdout: []string{
				"username=monalisa",
				"password=some-password",
			},
			wantStderr: "",
		},
		{
			name: "host only, no credentials found",
			opts: options{
				operation: "get",
				config: func() config.Config {
					return config.NewFromString(heredoc.Doc(`
						_source: "/Users/monalisa/.config/glab/config.yml"
						hosts:
						  example.com:
						    user: "monalisa"
					`))
				},
			},
			input: heredoc.Doc(`
				protocol=https
				host=example.com
			`),
			wantErr:         true,
			wantValidateErr: false,
			wantStderr:      "",
		},
		{
			name: "token from env",
			opts: options{
				operation: "get",
				config: func() config.Config {
					return config.NewFromString(heredoc.Doc(`
						_source: "GITLAB_TOKEN"
						hosts:
						  example.com:
						    user: "clemsbot"
						    token: "some-password"
					`))
				},
			},
			input: heredoc.Doc(`
				protocol=https
				host=example.com
				username=clemsbot
			`),
			wantErr:         false,
			wantValidateErr: false,
			wantStdout: []string{
				"username=clemsbot",
				"password=some-password",
			},
			wantStderr: "",
		},
		{
			name: "support OAuth2",
			opts: options{
				operation: "get",
				config: func() config.Config {
					return config.NewFromString(heredoc.Doc(`
						_source: "/Users/monalisa/.config/glab/config.yml"
						hosts:
						  example.com:
						    user: "monalisa"
						    is_oauth2: "true"
						    oauth2_refresh_token: "some-refresh-token"
						    oauth2_expiry_date: 13 Oct 25 12:35 UTC
						    token: "some-access-token"
					`))
				},
			},
			input: heredoc.Doc(`
				protocol=https
				host=example.com
			`),
			wantErr:         false,
			wantValidateErr: false,
			wantStdout: []string{
				"username=oauth2",
				"password=some-access-token",
				"password_expiry_utc=1760358900",
				"oauth_refresh_token=some-refresh-token",
			},
			wantStderr: "",
		},
		{
			name: "support Job Token",
			opts: options{
				operation: "get",
				config: func() config.Config {
					return config.NewFromString(heredoc.Doc(`
						_source: "/Users/monalisa/.config/glab/config.yml"
						hosts:
						  example.com:
						    user: "monalisa"
						    job_token: "some-job-token"
					`))
				},
			},
			input: heredoc.Doc(`
				protocol=https
				host=example.com
			`),
			wantErr:         false,
			wantValidateErr: false,
			wantStdout: []string{
				"username=gitlab-ci-token",
				"password=some-job-token",
			},
			wantStderr: "",
		},
		{
			name: "store command",
			opts: options{
				operation: "store",
				config: func() config.Config {
					return config.NewFromString(heredoc.Doc(`
						_source: "/Users/monalisa/.config/glab/config.yml"
						hosts:
						  example.com:
						    user: "monalisa"
						    token: "some-password"
					`))
				},
			},
			input: heredoc.Doc(`
				protocol=https
				host=example.com
			`),
			wantErr:         false,
			wantValidateErr: true,
			wantStdout:      nil,
			wantStderr:      "",
		},
		{
			name: "erase command",
			opts: options{
				operation: "erase",
				config: func() config.Config {
					return config.NewFromString(heredoc.Doc(`
						_source: "/Users/monalisa/.config/glab/config.yml"
						hosts:
						  example.com:
						    user: "monalisa"
						    token: "some-password"
					`))
				},
			},
			input: heredoc.Doc(`
				protocol=https
				host=example.com
			`),
			wantErr:         false,
			wantValidateErr: true,
			wantStdout:      nil,
			wantStderr:      "",
		},
		{
			name: "invalid command",
			opts: options{
				operation: "not-a-valid-command",
				config: func() config.Config {
					return config.NewFromString(heredoc.Doc(`
						_source: "/Users/monalisa/.config/glab/config.yml"
						hosts:
						  example.com:
						    user: "monalisa"
						    token: "some-password"
					`))
				},
			},
			input:           "",
			wantErr:         false,
			wantValidateErr: true,
			wantStdout:      nil,
			wantStderr:      "",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			io, stdin, stdout, stderr := cmdtest.TestIOStreams()
			fmt.Fprint(stdin, tt.input)
			opts := &tt.opts
			opts.io = io

			validateErr := opts.validate()
			if tt.wantValidateErr {
				assert.Error(t, validateErr)
			} else {
				assert.NoError(t, validateErr)
			}

			// No need to run if validate error is expected
			if tt.wantValidateErr {
				return
			}

			runErr := opts.run()
			if tt.wantErr {
				assert.Error(t, runErr)
			} else {
				assert.NoError(t, runErr)
			}

			if tt.wantStdout != nil {
				stdout := stdout.String()
				assert.Truef(t, strings.HasPrefix(stdout, "capability[]=authtype\n"), "first line of stdout must always be the capability preamble")
				t.Log(stdout)
				for _, expectedKV := range tt.wantStdout {
					assert.Contains(t, stdout, expectedKV)
				}
			}

			if tt.wantStderr != "" {
				assert.Equal(t, tt.wantStderr, stderr.String())
			}
		})
	}
}
