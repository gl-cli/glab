package login

import (
	"bufio"
	"fmt"
	"net/url"
	"strings"

	"gitlab.com/gitlab-org/cli/internal/cmdutils"
	"gitlab.com/gitlab-org/cli/internal/config"
	"gitlab.com/gitlab-org/cli/internal/iostreams"

	"github.com/spf13/cobra"
)

const tokenUser = "oauth2"

type options struct {
	io     *iostreams.IOStreams
	config func() config.Config

	operation string
}

func NewCmdCredential(f cmdutils.Factory) *cobra.Command {
	opts := &options{
		io:     f.IO(),
		config: f.Config,
	}

	cmd := &cobra.Command{
		Use:    "git-credential",
		Args:   cobra.ExactArgs(1),
		Short:  "Implements Git credential helper manager.",
		Hidden: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.complete(args)

			if err := opts.validate(); err != nil {
				return err
			}

			return opts.run()
		},
	}

	return cmd
}

func (o *options) complete(args []string) {
	o.operation = args[0]
}

func (o *options) validate() error {
	if o.operation == "store" {
		// We pretend to implement the "store" operation, but do nothing since we already have a cached token.
		return cmdutils.SilentError
	}

	if o.operation != "get" {
		return fmt.Errorf("glab auth git-credential: %q is an invalid operation.", o.operation)
	}

	return nil
}

func (o *options) run() error {
	expectedParams := map[string]string{}

	s := bufio.NewScanner(o.io.In)
	for s.Scan() {
		line := s.Text()
		if line == "" {
			break
		}
		parts := strings.SplitN(line, "=", 2)
		if len(parts) < 2 {
			continue
		}
		key, value := parts[0], parts[1]
		if key == "url" {
			u, err := url.Parse(value)
			if err != nil {
				return err
			}
			expectedParams["protocol"] = u.Scheme
			expectedParams["host"] = u.Host
			expectedParams["path"] = u.Path
			expectedParams["username"] = u.User.Username()
			expectedParams["password"], _ = u.User.Password()
		} else {
			expectedParams[key] = value
		}
	}
	if err := s.Err(); err != nil {
		return err
	}

	if expectedParams["protocol"] != "https" && expectedParams["protocol"] != "http" {
		return cmdutils.SilentError
	}

	cfg := o.config()

	gotToken, _ := cfg.Get(expectedParams["host"], "token")

	if gotToken == "" {
		return cmdutils.SilentError
	}

	fmt.Fprintf(o.io.StdOut, "protocol=%s\n", expectedParams["protocol"])
	fmt.Fprintf(o.io.StdOut, "host=%s\n", expectedParams["host"])
	fmt.Fprintf(o.io.StdOut, "username=%s\n", tokenUser)
	fmt.Fprintf(o.io.StdOut, "password=%s\n", gotToken)

	return nil
}
