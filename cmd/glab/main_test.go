package main

import (
	"bytes"
	"fmt"
	"net"
	"testing"

	"gitlab.com/gitlab-org/cli/pkg/iostreams"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"gitlab.com/gitlab-org/cli/commands/cmdutils"
	"go.uber.org/goleak"
)

func Test_printError(t *testing.T) {
	cmd := &cobra.Command{}

	type args struct {
		err   error
		cmd   *cobra.Command
		debug bool
	}
	tests := []struct {
		name    string
		args    args
		wantOut string
	}{
		{
			name: "generic error",
			args: args{
				err:   errors.New("the app exploded"),
				cmd:   cmd,
				debug: false,
			},
			wantOut: "the app exploded\n",
		},
		{
			name: "DNS error",
			args: args{
				err: fmt.Errorf("DNS error: %w", &net.DNSError{
					Name: "https://gitlab.com/api/v4",
				}),
				cmd:   cmd,
				debug: false,
			},
			wantOut: `x error connecting to https://gitlab.com/api/v4
• Check your internet connection and status.gitlab.com. If on a self-managed instance, run 'sudo gitlab-ctl status' on your server.
`,
		},
		{
			name: "DNS error with debug",
			args: args{
				err: fmt.Errorf("DNS error: %w", &net.DNSError{
					Name: "https://gitlab.com/api/v4",
				}),
				cmd:   cmd,
				debug: true,
			},

			wantOut: `x error connecting to https://gitlab.com/api/v4
x lookup https://gitlab.com/api/v4: 
• Check your internet connection and status.gitlab.com. If on a self-managed instance, run 'sudo gitlab-ctl status' on your server.
`,
		},
		{
			name: "Cobra flag error",
			args: args{
				err:   &cmdutils.FlagError{Err: errors.New("unknown flag --foo")},
				cmd:   cmd,
				debug: false,
			},
			wantOut: "unknown flag --foo\n\nUsage:\n\n",
		},
		{
			name: "unknown Cobra command error",
			args: args{
				err:   errors.New("unknown command foo"),
				cmd:   cmd,
				debug: false,
			},
			wantOut: "unknown command foo\n\nUsage:\n\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			streams, _, _, _ := iostreams.Test()
			out := &bytes.Buffer{}
			streams.StdErr = out
			printError(streams, tt.args.err, tt.args.cmd, tt.args.debug, false)
			if gotOut := out.String(); gotOut != tt.wantOut {
				t.Errorf("printError() = %q, want %q", gotOut, tt.wantOut)
			}
		})
	}
}

// Test started when the test binary is started
// and calls the main function
func TestGlab(t *testing.T) {
	main()
}

func TestMain(m *testing.M) {
	goleak.VerifyTestMain(m)
}
