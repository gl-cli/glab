package cmdutils

import (
	"errors"
	"fmt"
	"io"

	"github.com/AlecAivazis/survey/v2/terminal"
	"github.com/charmbracelet/fang"
	"github.com/spf13/cobra"
)

// FlagError is the kind of error raised in flag processing
type FlagError struct {
	Err error
}

func (fe FlagError) Error() string {
	return fe.Err.Error()
}

func (fe FlagError) Unwrap() error {
	return fe.Err
}

// SilentError is an error that triggers exit Code 1 without any error messaging
var SilentError = errors.New("SilentError")

// GitLabErrorHandler is a custom error handler for fang that handles GitLab CLI specific errors
func GitLabErrorHandler(w io.Writer, styles fang.Styles, err error) {
	// Ignore SilentError - it should not produce any output
	if errors.Is(err, SilentError) {
		return
	}
	// Delegate everything else to Fang's default handler
	fang.DefaultErrorHandler(w, styles, err)
}

type ExitError struct {
	Err     error
	Code    int
	Details string
}

func WrapErrorWithCode(err error, code int, details string) *ExitError {
	return &ExitError{
		Err:     err,
		Code:    code,
		Details: details,
	}
}

func WrapError(err error, log string) *ExitError {
	return WrapErrorWithCode(err, 1, log)
}

func CancelError(log ...any) error {
	if len(log) < 1 {
		return WrapErrorWithCode(terminal.InterruptErr, 2, "action cancelled")
	}
	return WrapErrorWithCode(terminal.InterruptErr, 2, fmt.Sprint(log...))
}

func (e *ExitError) Error() string {
	return e.Err.Error()
}

func (e ExitError) Unwrap() error {
	return e.Err
}

func MinimumArgs(n int, msg string) cobra.PositionalArgs {
	if msg == "" {
		return cobra.MinimumNArgs(1)
	}

	return func(cmd *cobra.Command, args []string) error {
		if len(args) < n {
			return &FlagError{Err: errors.New(msg)}
		}
		return nil
	}
}
