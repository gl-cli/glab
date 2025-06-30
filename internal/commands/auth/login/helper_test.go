package login

import (
	"fmt"
	"testing"

	"gitlab.com/gitlab-org/cli/internal/config"
	"gitlab.com/gitlab-org/cli/internal/testing/cmdtest"

	"github.com/MakeNowJust/heredoc/v2"
)

func Test_helperRun(t *testing.T) {
	// NOTE: we have to remove the values for the possible token env variables, because it'll take precedence over the config.
	// See config.EnvKeyEquivalence function.
	t.Setenv("GITLAB_TOKEN", "")
	t.Setenv("GITLAB_ACCESS_TOKEN", "")
	t.Setenv("OAUTH_TOKEN", "")

	tests := []struct {
		name       string
		opts       options
		input      string
		wantStdout string
		wantStderr string
		wantErr    bool
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
						    token: "OTOKEN"
					`))
				},
			},
			input: heredoc.Doc(`
				protocol=https
				host=example.com
			`),
			wantErr: false,
			wantStdout: heredoc.Doc(`
				protocol=https
				host=example.com
				username=oauth2
				password=OTOKEN
			`),
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
						    token: "OTOKEN"
					`))
				},
			},
			input: heredoc.Doc(`
				protocol=https
				host=example.com
				username=monalisa
			`),
			wantErr: false,
			wantStdout: heredoc.Doc(`
				protocol=https
				host=example.com
				username=oauth2
				password=OTOKEN
			`),
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
						    token: "OTOKEN"
					`))
				},
			},
			input: heredoc.Doc(`
				url=https://monalisa@example.com
			`),
			wantErr: false,
			wantStdout: heredoc.Doc(`
				protocol=https
				host=example.com
				username=oauth2
				password=OTOKEN
			`),
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
			wantErr:    true,
			wantStdout: "",
			wantStderr: "",
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
						    token: "OTOKEN"
					`))
				},
			},
			input: heredoc.Doc(`
				protocol=https
				host=example.com
				username=clemsbot
			`),
			wantErr: false,
			wantStdout: heredoc.Doc(`
				protocol=https
				host=example.com
				username=oauth2
				password=OTOKEN
			`),
			wantStderr: "",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			io, stdin, stdout, stderr := cmdtest.TestIOStreams()
			fmt.Fprint(stdin, tt.input)
			opts := &tt.opts
			opts.io = io
			if err := opts.run(); (err != nil) != tt.wantErr {
				t.Fatalf("helperRun() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantStdout != stdout.String() {
				t.Errorf("stdout: got %q, wants %q", stdout.String(), tt.wantStdout)
			}
			if tt.wantStderr != stderr.String() {
				t.Errorf("stderr: got %q, wants %q", stderr.String(), tt.wantStderr)
			}
		})
	}
}
