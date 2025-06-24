package login

import (
	"fmt"
	"testing"

	"gitlab.com/gitlab-org/cli/pkg/iostreams"

	"github.com/MakeNowJust/heredoc/v2"
)

type tinyConfig map[string]string

func (c tinyConfig) Get(host, key string) (string, error) {
	return c[fmt.Sprintf("%s:%s", host, key)], nil
}

func Test_helperRun(t *testing.T) {
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
				config: func() (configExt, error) {
					return tinyConfig{
						"_source":           "/Users/monalisa/.config/glab/config.yml",
						"example.com:user":  "monalisa",
						"example.com:token": "OTOKEN",
					}, nil
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
				config: func() (configExt, error) {
					return tinyConfig{
						"_source":           "/Users/monalisa/.config/glab/config.yml",
						"example.com:user":  "monalisa",
						"example.com:token": "OTOKEN",
					}, nil
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
				config: func() (configExt, error) {
					return tinyConfig{
						"_source":           "/Users/monalisa/.config/glab/config.yml",
						"example.com:user":  "monalisa",
						"example.com:token": "OTOKEN",
					}, nil
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
				config: func() (configExt, error) {
					return tinyConfig{
						"_source":          "/Users/monalisa/.config/glab/config.yml",
						"example.com:user": "monalisa",
					}, nil
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
				config: func() (configExt, error) {
					return tinyConfig{
						"_source":           "GITLAB_TOKEN",
						"example.com:token": "OTOKEN",
					}, nil
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
			io, stdin, stdout, stderr := iostreams.Test()
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
